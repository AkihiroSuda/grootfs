package groot

import (
	"net/url"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	errorspkg "github.com/pkg/errors"
)

const ImageReferenceFormat = "image:%s"

type CreateSpec struct {
	ID                          string
	BaseImage                   string
	DiskLimit                   int64
	Mount                       bool
	ExcludeBaseImageFromQuota   bool
	CleanOnCreate               bool
	CleanOnCreateThresholdBytes int64
	CleanOnCreateIgnoreImages   []string
	UIDMappings                 []IDMappingSpec
	GIDMappings                 []IDMappingSpec
}

type Creator struct {
	cleaner           Cleaner
	imageCloner       ImageCloner
	baseImagePuller   BaseImagePuller
	locksmith         Locksmith
	dependencyManager DependencyManager
	metricsEmitter    MetricsEmitter
}

func IamCreator(
	imageCloner ImageCloner, baseImagePuller BaseImagePuller,
	locksmith Locksmith, dependencyManager DependencyManager,
	metricsEmitter MetricsEmitter, cleaner Cleaner) *Creator {
	return &Creator{
		imageCloner:       imageCloner,
		baseImagePuller:   baseImagePuller,
		locksmith:         locksmith,
		dependencyManager: dependencyManager,
		metricsEmitter:    metricsEmitter,
		cleaner:           cleaner,
	}
}

func (c *Creator) Create(logger lager.Logger, spec CreateSpec) (ImageInfo, error) {
	defer c.metricsEmitter.TryEmitDurationFrom(logger, MetricImageCreationTime, time.Now())

	logger = logger.Session("groot-creating", lager.Data{"imageID": spec.ID, "spec": spec})
	logger.Info("starting")
	defer logger.Info("ending")

	parsedURL, err := url.Parse(spec.BaseImage)
	if err != nil {
		return ImageInfo{}, errorspkg.Wrap(err, "parsing image url")
	}

	if strings.ContainsAny(spec.ID, "/") {
		return ImageInfo{}, errorspkg.Errorf("id `%s` contains invalid characters: `/`", spec.ID)
	}

	ok, err := c.imageCloner.Exists(spec.ID)
	if err != nil {
		return ImageInfo{}, errorspkg.Wrap(err, "checking id exists")
	}
	if ok {
		return ImageInfo{}, errorspkg.Errorf("image for id `%s` already exists", spec.ID)
	}

	ownerUid, ownerGid := c.parseOwner(spec.UIDMappings, spec.GIDMappings)
	baseImageSpec := BaseImageSpec{
		BaseImageSrc:              parsedURL,
		DiskLimit:                 spec.DiskLimit,
		ExcludeBaseImageFromQuota: spec.ExcludeBaseImageFromQuota,
		UIDMappings:               spec.UIDMappings,
		GIDMappings:               spec.GIDMappings,
		OwnerUID:                  ownerUid,
		OwnerGID:                  ownerGid,
	}

	if spec.CleanOnCreate {
		ignoredImages := append(spec.CleanOnCreateIgnoreImages, spec.BaseImage)
		if _, err := c.cleaner.Clean(logger, spec.CleanOnCreateThresholdBytes, ignoredImages); err != nil {
			return ImageInfo{}, errorspkg.Wrap(err, "failed-to-cleanup-store")
		}
	}

	lockFile, err := c.locksmith.Lock(GlobalLockKey)
	if err != nil {
		return ImageInfo{}, err
	}
	defer func() {
		if err := c.locksmith.Unlock(lockFile); err != nil {
			logger.Error("failed-to-unlock", err)
		}
	}()

	baseImage, err := c.baseImagePuller.Pull(logger, baseImageSpec)
	if err != nil {
		return ImageInfo{}, errorspkg.Wrap(err, "pulling the image")
	}

	imageSpec := ImageSpec{
		ID:                        spec.ID,
		Mount:                     spec.Mount,
		DiskLimit:                 spec.DiskLimit,
		ExcludeBaseImageFromQuota: spec.ExcludeBaseImageFromQuota,
		BaseVolumeIDs:             baseImage.ChainIDs,
		BaseImage:                 baseImage.BaseImage,
		OwnerUID:                  ownerUid,
		OwnerGID:                  ownerGid,
	}

	image, err := c.imageCloner.Create(logger, imageSpec)
	if err != nil {
		return ImageInfo{}, errorspkg.Wrap(err, "making image")
	}

	// Register all volumes that are used in this image
	//
	// New Scheme:
	// Pulling layer ABC and laying it in /volumes/ABC should result in a /meta/volumes/ABC-ref-counter file
	// We should create /images/ABC-ref hard link pointing to /meta/volumes/ABC-ref-counter
	if err := c.dependencyManager.Register(spec.ID, baseImage.ChainIDs); err != nil {
		if destroyErr := c.imageCloner.Destroy(logger, spec.ID); destroyErr != nil {
			logger.Error("failed-to-destroy-image", destroyErr)
		}

		return ImageInfo{}, err
	}

	return image, nil
}

func (c *Creator) parseOwner(uidMappings, gidMappings []IDMappingSpec) (int, int) {
	uid := os.Getuid()
	gid := os.Getgid()

	for _, mapping := range uidMappings {
		if mapping.Size == 1 && mapping.NamespaceID == 0 {
			uid = mapping.HostID
			break
		}
	}

	for _, mapping := range gidMappings {
		if mapping.Size == 1 && mapping.NamespaceID == 0 {
			gid = mapping.HostID
			break
		}
	}

	return uid, gid
}
