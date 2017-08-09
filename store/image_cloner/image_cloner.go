package image_cloner // import "code.cloudfoundry.org/grootfs/store/image_cloner"

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/grootfs/store"
	"code.cloudfoundry.org/lager"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	errorspkg "github.com/pkg/errors"
)

type ImageDriverSpec struct {
	BaseVolumeIDs      []string
	Mount              bool
	ImagePath          string
	DiskLimit          int64
	ExclusiveDiskLimit bool
}

//go:generate counterfeiter . ImageDriver
type ImageDriver interface {
	CreateImage(logger lager.Logger, spec ImageDriverSpec) (groot.MountInfo, error)
	DestroyImage(logger lager.Logger, path string) error
	FetchStats(logger lager.Logger, path string) (groot.VolumeStats, error)
}

type ImageCloner struct {
	imageDriver ImageDriver
	storePath   string
}

func NewImageCloner(imageDriver ImageDriver, storePath string) *ImageCloner {
	return &ImageCloner{
		imageDriver: imageDriver,
		storePath:   storePath,
	}
}

func (b *ImageCloner) ImageIDs(logger lager.Logger) ([]string, error) {
	images := []string{}

	existingImages, err := ioutil.ReadDir(path.Join(b.storePath, store.ImageDirName))
	if err != nil {
		return nil, errorspkg.Wrap(err, "failed to read images dir")
	}

	for _, imageInfo := range existingImages {
		images = append(images, imageInfo.Name())
	}

	return images, nil
}

func (b *ImageCloner) Create(logger lager.Logger, spec groot.ImageSpec) (groot.ImageInfo, error) {
	logger = logger.Session("making-image", lager.Data{"storePath": b.storePath, "id": spec.ID})
	logger.Info("starting")
	defer logger.Info("ending")

	imagePath := b.imagePath(spec.ID)
	imageRootFSPath := filepath.Join(imagePath, "rootfs")

	var err error
	defer func() {
		if err != nil {
			log := logger.Session("create-failed-cleaning-up", lager.Data{
				"id":    spec.ID,
				"cause": err.Error(),
			})

			log.Info("starting")
			defer log.Info("ending")

			if err = b.imageDriver.DestroyImage(logger, imagePath); err != nil {
				log.Error("destroying-rootfs-image", err)
			}

			if err = b.deleteImageDir(imagePath); err != nil {
				log.Error("deleting-image-path", err)
			}
		}
	}()

	if err = os.Mkdir(imagePath, 0700); err != nil {
		return groot.ImageInfo{}, errorspkg.Wrap(err, "making image path")
	}

	if err = b.writeBaseImageJSON(logger, imagePath, spec.BaseImage); err != nil {
		logger.Error("writing-image-json-failed", err)
		return groot.ImageInfo{}, errorspkg.Wrap(err, "creating image.json")
	}

	imageDriverSpec := ImageDriverSpec{
		BaseVolumeIDs:      spec.BaseVolumeIDs,
		Mount:              spec.Mount,
		ImagePath:          imagePath,
		DiskLimit:          spec.DiskLimit,
		ExclusiveDiskLimit: spec.ExcludeBaseImageFromQuota,
	}

	var mountInfo groot.MountInfo
	if mountInfo, err = b.imageDriver.CreateImage(logger, imageDriverSpec); err != nil {
		logger.Error("creating-image-failed", err, lager.Data{"imageDriverSpec": imageDriverSpec})
		return groot.ImageInfo{}, errorspkg.Wrap(err, "creating image")
	}

	if err := b.setOwnership(spec,
		imagePath,
		filepath.Join(imagePath, "image.json"),
		imageRootFSPath,
	); err != nil {
		logger.Error("setting-permission-failed", err, lager.Data{"imageDriverSpec": imageDriverSpec})
		return groot.ImageInfo{}, err
	}

	imageInfo, err := b.imageInfo(imageRootFSPath, imagePath, spec.BaseImage, mountInfo, spec.Mount)
	if err != nil {
		logger.Error("creating-image-object", err)
		return groot.ImageInfo{}, errorspkg.Wrap(err, "creating image object")
	}

	return imageInfo, nil
}

func (b *ImageCloner) Destroy(logger lager.Logger, id string) error {
	logger = logger.Session("deleting-image", lager.Data{"storePath": b.storePath, "id": id})
	logger.Info("starting")
	defer logger.Info("ending")

	if ok, err := b.Exists(id); !ok {
		logger.Error("checking-image-path-failed", err)
		if err != nil {
			return errorspkg.Wrapf(err, "unable to check image: %s", id)
		}
		return errorspkg.Errorf("image not found: %s", id)
	}

	imagePath := b.imagePath(id)
	volDriverErr := b.imageDriver.DestroyImage(logger, imagePath)
	if volDriverErr != nil {
		logger.Error("destroying-image-failed", volDriverErr)
	}

	if err := b.deleteImageDir(imagePath); err != nil {
		logger.Error("deleting-image-dir-failed", err, lager.Data{"volumeDriverError": volDriverErr})
		return errorspkg.Wrap(err, "deleting image path")
	}

	return nil
}

func (b *ImageCloner) Exists(id string) (bool, error) {
	imagePath := path.Join(b.storePath, store.ImageDirName, id)
	if _, err := os.Stat(imagePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errorspkg.Wrapf(err, "checking if image `%s` exists", id)
	}

	return true, nil
}

func (b *ImageCloner) Stats(logger lager.Logger, id string) (groot.VolumeStats, error) {
	logger = logger.Session("fetching-stats", lager.Data{"id": id})
	logger.Debug("starting")
	defer logger.Debug("ending")

	if ok, err := b.Exists(id); !ok {
		logger.Error("checking-image-path-failed", err)
		return groot.VolumeStats{}, errorspkg.Errorf("image not found: %s", id)
	}

	imagePath := b.imagePath(id)

	return b.imageDriver.FetchStats(logger, imagePath)
}

func (b *ImageCloner) deleteImageDir(imagePath string) error {
	if err := os.RemoveAll(imagePath); err != nil {
		return errorspkg.Wrap(err, "deleting image path")
	}

	return nil
}

var OF = os.OpenFile

func (b *ImageCloner) writeBaseImageJSON(logger lager.Logger, imagePath string, baseImage *specsv1.Image) error {
	logger = logger.Session("writing-image-json")
	logger.Debug("starting")
	defer logger.Debug("ending")

	imageJsonPath := filepath.Join(imagePath, "image.json")
	imageJsonFile, err := OF(imageJsonPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	if err = json.NewEncoder(imageJsonFile).Encode(baseImage); err != nil {
		return err
	}

	return nil
}

func (b *ImageCloner) imageInfo(rootfsPath, imagePath string, baseImage *specsv1.Image, mountJson groot.MountInfo, mount bool) (groot.ImageInfo, error) {
	imageInfo := groot.ImageInfo{
		Path:   imagePath,
		Rootfs: rootfsPath,
		Image:  baseImage,
	}

	if !mount {
		imageInfo.Mount = &mountJson
	}

	return imageInfo, nil
}

func (b *ImageCloner) imagePath(id string) string {
	return path.Join(b.storePath, store.ImageDirName, id)
}

func (b *ImageCloner) setOwnership(spec groot.ImageSpec, paths ...string) error {
	if spec.OwnerUID == 0 && spec.OwnerGID == 0 {
		return nil
	}

	for _, path := range paths {
		if err := os.Chown(path, spec.OwnerUID, spec.OwnerGID); err != nil {
			return errorspkg.Wrapf(err, "changing %s ownership to %d:%d", path, spec.OwnerUID, spec.OwnerGID)
		}
	}
	return nil
}
