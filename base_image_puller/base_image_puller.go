package base_image_puller // import "code.cloudfoundry.org/grootfs/base_image_puller"

import (
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/lager"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	errorspkg "github.com/pkg/errors"
)

const BaseImageReferenceFormat = "baseimage:%s"

//go:generate counterfeiter . Fetcher
//go:generate counterfeiter . Unpacker
//go:generate counterfeiter . DependencyRegisterer
//go:generate counterfeiter . VolumeDriver

type UnpackSpec struct {
	Stream      io.ReadCloser
	TargetPath  string
	UIDMappings []groot.IDMappingSpec
	GIDMappings []groot.IDMappingSpec
}

type LayerDigest struct {
	BlobID        string
	ChainID       string
	ParentChainID string
	Size          int64
}

type BaseImageInfo struct {
	LayersDigest []LayerDigest
	Config       specsv1.Image
}

type Fetcher interface {
	BaseImageInfo(logger lager.Logger, baseImageURL *url.URL) (BaseImageInfo, error)
	StreamBlob(logger lager.Logger, baseImageURL *url.URL, source string) (io.ReadCloser, int64, error)
}

type DependencyRegisterer interface {
	Register(id string, chainIDs []string) error
}

type Unpacker interface {
	Unpack(logger lager.Logger, spec UnpackSpec) error
}

type VolumeDriver interface {
	VolumePath(logger lager.Logger, id string) (string, error)
	CreateVolume(logger lager.Logger, parentID, id string) (string, error)
	DestroyVolume(logger lager.Logger, id string) error
	Volumes(logger lager.Logger) ([]string, error)
	MoveVolume(from, to string) error
}

type BaseImagePuller struct {
	localFetcher         Fetcher
	remoteFetcher        Fetcher
	unpacker             Unpacker
	volumeDriver         VolumeDriver
	dependencyRegisterer DependencyRegisterer
}

func NewBaseImagePuller(localFetcher, remoteFetcher Fetcher, unpacker Unpacker, volumeDriver VolumeDriver, dependencyRegisterer DependencyRegisterer) *BaseImagePuller {
	return &BaseImagePuller{
		localFetcher:         localFetcher,
		remoteFetcher:        remoteFetcher,
		unpacker:             unpacker,
		volumeDriver:         volumeDriver,
		dependencyRegisterer: dependencyRegisterer,
	}
}

func (p *BaseImagePuller) Pull(logger lager.Logger, spec groot.BaseImageSpec) (groot.BaseImage, error) {
	logger = logger.Session("image-pulling", lager.Data{"spec": spec})
	logger.Info("start")
	defer logger.Info("end")
	var err error

	baseImageInfo, err := p.fetcher(spec.BaseImageSrc).BaseImageInfo(logger, spec.BaseImageSrc)
	if err != nil {
		return groot.BaseImage{}, errorspkg.Wrap(err, "fetching list of digests")
	}
	logger.Debug("fetched-layers-digests", lager.Data{"digests": baseImageInfo.LayersDigest})

	if err := p.quotaExceeded(logger, baseImageInfo.LayersDigest, spec); err != nil {
		return groot.BaseImage{}, err
	}

	volumePath, err := p.buildLayer(logger, len(baseImageInfo.LayersDigest)-1, baseImageInfo.LayersDigest, spec)
	if err != nil {
		return groot.BaseImage{}, err
	}
	chainIDs := p.chainIDs(baseImageInfo.LayersDigest)

	baseImageRefName := fmt.Sprintf(BaseImageReferenceFormat, spec.BaseImageSrc.String())
	if err := p.dependencyRegisterer.Register(baseImageRefName, chainIDs); err != nil {
		return groot.BaseImage{}, err
	}

	baseImage := groot.BaseImage{
		BaseImage:  baseImageInfo.Config,
		ChainIDs:   chainIDs,
		VolumePath: volumePath,
	}
	return baseImage, nil
}

func (p *BaseImagePuller) fetcher(baseImageURL *url.URL) Fetcher {
	if baseImageURL.Scheme == "" {
		return p.localFetcher
	} else {
		return p.remoteFetcher
	}
}

func (p *BaseImagePuller) quotaExceeded(logger lager.Logger, layersDigest []LayerDigest, spec groot.BaseImageSpec) error {
	if spec.ExcludeBaseImageFromQuota || spec.DiskLimit == 0 {
		return nil
	}

	totalSize := p.layersSize(layersDigest)
	if totalSize > spec.DiskLimit {
		err := errorspkg.Errorf("layers exceed disk quota %d/%d bytes", totalSize, spec.DiskLimit)
		logger.Error("blob-manifest-size-check-failed", err, lager.Data{
			"totalSize":                 totalSize,
			"diskLimit":                 spec.DiskLimit,
			"excludeBaseImageFromQuota": spec.ExcludeBaseImageFromQuota,
		})
		return err
	}

	return nil
}

func (p *BaseImagePuller) chainIDs(layersDigest []LayerDigest) []string {
	chainIDs := []string{}
	for _, layerDigest := range layersDigest {
		chainIDs = append(chainIDs, layerDigest.ChainID)
	}
	return chainIDs
}

func (p *BaseImagePuller) buildLayer(logger lager.Logger, index int, layersDigest []LayerDigest, spec groot.BaseImageSpec) (string, error) {
	if index < 0 {
		return "", nil
	}

	digest := layersDigest[index]
	volumePath, err := p.volumeDriver.VolumePath(logger, digest.ChainID)
	if err == nil {
		logger.Debug("volume-exists", lager.Data{
			"volumePath":    volumePath,
			"blobID":        digest.BlobID,
			"chainID":       digest.ChainID,
			"parentChainID": digest.ParentChainID,
		})
		return volumePath, nil
	}

	if _, err := p.buildLayer(logger, index-1, layersDigest, spec); err != nil {
		return "", err
	}

	stream, size, err := p.fetcher(spec.BaseImageSrc).StreamBlob(logger, spec.BaseImageSrc, digest.BlobID)
	if err != nil {
		return "", errorspkg.Wrapf(err, "streaming blob `%s`", digest.BlobID)
	}
	defer stream.Close()

	logger.Debug("got-stream-for-blob", lager.Data{
		"size":                      size,
		"diskLimit":                 spec.DiskLimit,
		"excludeBaseImageFromQuota": spec.ExcludeBaseImageFromQuota,
		"blobID":                    digest.BlobID,
		"chainID":                   digest.ChainID,
		"parentChainID":             digest.ParentChainID,
	})

	tempVolumeName := fmt.Sprintf("%s-%d-%d", digest.ChainID, time.Now().Unix(), rand.Int())
	volumePath, err = p.volumeDriver.CreateVolume(logger,
		digest.ParentChainID,
		tempVolumeName,
	)
	if err != nil {
		return "", errorspkg.Wrapf(err, "creating volume for layer `%s`", digest.BlobID)
	}
	logger.Debug("volume-created", lager.Data{
		"volumePath":    volumePath,
		"blobID":        digest.BlobID,
		"chainID":       digest.ChainID,
		"parentChainID": digest.ParentChainID,
	})

	if spec.OwnerUID != 0 || spec.OwnerGID != 0 {
		err = os.Chown(volumePath, spec.OwnerUID, spec.OwnerGID)
		if err != nil {
			return "", errorspkg.Wrapf(err, "changing volume ownership to %d:%d", spec.OwnerUID, spec.OwnerGID)
		}
	}

	unpackSpec := UnpackSpec{
		TargetPath:  volumePath,
		Stream:      stream,
		UIDMappings: spec.UIDMappings,
		GIDMappings: spec.GIDMappings,
	}

	if err := p.unpacker.Unpack(logger, unpackSpec); err != nil {
		if errD := p.volumeDriver.DestroyVolume(logger, digest.ChainID); errD != nil {
			logger.Error("volume-cleanup-failed", errD, lager.Data{
				"blobID":        digest.BlobID,
				"chainID":       digest.ChainID,
				"parentChainID": digest.ParentChainID,
			})
		}
		return "", errorspkg.Wrapf(err, "unpacking layer `%s`", digest.BlobID)
	}
	logger.Debug("layer-unpacked", lager.Data{
		"blobID":        digest.BlobID,
		"chainID":       digest.ChainID,
		"parentChainID": digest.ParentChainID,
	})

	finalVolumePath := strings.Replace(volumePath, tempVolumeName, digest.ChainID, 1)
	if err := p.volumeDriver.MoveVolume(volumePath, finalVolumePath); err != nil {
		logger.Error("moving-volume-failed", err, lager.Data{"from": volumePath, "to": finalVolumePath})
		return "", err
	}
	return finalVolumePath, nil
}

func (p *BaseImagePuller) layersSize(layerDigests []LayerDigest) int64 {
	var totalSize int64
	for _, digest := range layerDigests {
		totalSize += digest.Size
	}
	return totalSize
}
