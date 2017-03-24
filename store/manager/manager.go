package manager

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/grootfs/base_image_puller"
	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/grootfs/store"
	"code.cloudfoundry.org/grootfs/store/image_cloner"
	"code.cloudfoundry.org/lager"
	errorspkg "github.com/pkg/errors"
)

//go:generate counterfeiter . StoreDriver
type StoreDriver interface {
	ConfigureStore(logger lager.Logger, storePath string, ownerUID, ownerGID int) error
	ValidateFileSystem(logger lager.Logger, path string) error
}

type Manager struct {
	storePath    string
	imageDriver  image_cloner.ImageDriver
	volumeDriver base_image_puller.VolumeDriver
	storeDriver  StoreDriver
	locksmith    groot.Locksmith
}

func New(storePath string, locksmith groot.Locksmith, volumeDriver base_image_puller.VolumeDriver, imageDriver image_cloner.ImageDriver, storeDriver StoreDriver) *Manager {
	return &Manager{
		storePath:    storePath,
		volumeDriver: volumeDriver,
		imageDriver:  imageDriver,
		storeDriver:  storeDriver,
		locksmith:    locksmith,
	}
}

func (m *Manager) InitStore(logger lager.Logger) error {
	logger = logger.Session("store-manager-init-store")
	logger.Debug("starting")
	defer logger.Debug("ending")

	stat, err := os.Stat(m.storePath)
	if err == nil && stat.IsDir() {
		logger.Error("store-path-already-exists", errorspkg.Errorf("%s already exists", m.storePath))
		return errorspkg.Errorf("store already initialized at path %s", m.storePath)
	}

	if err := m.storeDriver.ValidateFileSystem(logger, filepath.Dir(m.storePath)); err != nil {
		logger.Error("store-path-validation-failed", err)
		return errorspkg.Wrap(err, "validating store path filesystem")
	}

	if err := os.MkdirAll(m.storePath, 0755); err != nil {
		logger.Error("init-store-failed", err, lager.Data{"storePath": m.storePath})
		return errorspkg.Wrap(err, "initializing store")
	}
	return nil
}

func (m *Manager) ConfigureStore(logger lager.Logger, ownerUID, ownerGID int) error {
	logger = logger.Session("store-manager-configure-store", lager.Data{"storePath": m.storePath, "ownerUID": ownerUID, "ownerGID": ownerGID})
	logger.Debug("starting")
	defer logger.Debug("ending")

	if err := isDirectory(m.storePath); err != nil {
		return err
	}

	if err := os.Setenv("TMPDIR", filepath.Join(m.storePath, store.TempDirName)); err != nil {
		return errorspkg.Wrap(err, "could not set TMPDIR")
	}

	requiredFolders := []string{
		store.ImageDirName,
		store.VolumesDirName,
		store.CacheDirName,
		store.LocksDirName,
		store.MetaDirName,
		store.TempDirName,
		filepath.Join(store.MetaDirName, "dependencies"),
	}

	if _, err := os.Stat(m.storePath); os.IsNotExist(err) {
		if err := os.Mkdir(m.storePath, 0755); err != nil {
			dir, err1 := os.Lstat(m.storePath)
			if err1 != nil || !dir.IsDir() {
				logger.Error("creating-store-path-failed", err)
				return errorspkg.Wrapf(err, "making directory `%s`", m.storePath)
			}
		}

		if err := os.Chown(m.storePath, ownerUID, ownerGID); err != nil {
			logger.Error("store-ownership-change-failed", err, lager.Data{"target-uid": ownerUID, "target-gid": ownerGID})
			return errorspkg.Wrapf(err, "changing store owner to %d:%d for path %s", ownerUID, ownerGID, m.storePath)
		}

		if err := os.Chmod(m.storePath, 0700); err != nil {
			logger.Error("store-permission-change-failed", err)
			return errorspkg.Wrapf(err, "changing store permissions %s", m.storePath)
		}
	}

	for _, folderName := range requiredFolders {
		if err := m.createInternalDirectory(logger, folderName, ownerUID, ownerGID); err != nil {
			return err
		}
	}

	if err := m.storeDriver.ValidateFileSystem(logger, m.storePath); err != nil {
		logger.Error("filesystem-validation-failed", err)
		return errorspkg.Wrap(err, "validating file system")
	}

	if err := m.storeDriver.ConfigureStore(logger, m.storePath, ownerUID, ownerGID); err != nil {
		logger.Error("store-filesystem-specific-configuration-failed", err)
		return errorspkg.Wrap(err, "running filesystem-specific configuration")
	}

	return nil
}

func (m *Manager) DeleteStore(logger lager.Logger) error {
	logger = logger.Session("store-manager-delete-store")
	logger.Debug("starting")
	defer logger.Debug("ending")

	fileLock, err := m.locksmith.Lock(groot.GlobalLockKey)
	if err != nil {
		logger.Error("locking-failed", err)
		return errorspkg.Wrap(err, "requesting lock")
	}
	defer m.locksmith.Unlock(fileLock)

	existingImages, err := m.images()
	if err != nil {
		return err
	}

	for _, image := range existingImages {
		if err := m.imageDriver.DestroyImage(logger, image); err != nil {
			logger.Error("destroing-image-failed", err, lager.Data{"image": image})
			return errorspkg.Wrapf(err, "destroying image %s", image)
		}
	}

	existingVolumes, err := m.volumes()
	if err != nil {
		return err
	}

	for _, volume := range existingVolumes {
		if err := m.volumeDriver.DestroyVolume(logger, volume); err != nil {
			logger.Error("destroing-volume-failed", err, lager.Data{"volume": volume})
			return errorspkg.Wrapf(err, "destroying volume %s", volume)
		}
	}

	if err := os.RemoveAll(m.storePath); err != nil {
		logger.Error("deleting-store-path-failed", err, lager.Data{"storePath": m.storePath})
		return errorspkg.Wrapf(err, "deleting store path")
	}

	return nil
}

func (m *Manager) images() ([]string, error) {
	imagesPath := filepath.Join(m.storePath, store.ImageDirName)
	images, err := ioutil.ReadDir(imagesPath)
	if err != nil {
		return nil, errorspkg.Wrap(err, "listing images")
	}

	imagePaths := []string{}
	for _, file := range images {
		imagePaths = append(imagePaths, filepath.Join(imagesPath, file.Name()))
	}

	return imagePaths, nil
}

func (m *Manager) volumes() ([]string, error) {
	volumesPath := filepath.Join(m.storePath, store.VolumesDirName)
	volumes, err := ioutil.ReadDir(volumesPath)
	if err != nil {
		return nil, errorspkg.Wrap(err, "listing volumes")
	}

	volumeIds := []string{}
	for _, file := range volumes {
		volumeIds = append(volumeIds, file.Name())
	}

	return volumeIds, nil
}

func (m *Manager) createInternalDirectory(logger lager.Logger, folderName string, ownerUID, ownerGID int) error {
	requiredPath := filepath.Join(m.storePath, folderName)

	if err := isDirectory(requiredPath); err != nil {
		return err
	}

	if err := os.Mkdir(requiredPath, 0755); err != nil {
		dir, err1 := os.Lstat(requiredPath)
		if err1 != nil || !dir.IsDir() {
			return errorspkg.Wrapf(err, "making directory `%s`", requiredPath)
		}
	}

	if err := os.Chown(requiredPath, ownerUID, ownerGID); err != nil {
		logger.Error("store-ownership-change-failed", err, lager.Data{"target-uid": ownerUID, "target-gid": ownerGID})
		return errorspkg.Wrapf(err, "changing store owner to %d:%d for path %s", ownerUID, ownerGID, requiredPath)
	}
	return nil
}

func isDirectory(requiredPath string) error {
	if info, err := os.Stat(requiredPath); err == nil {
		if !info.IsDir() {
			return errorspkg.Errorf("path `%s` is not a directory", requiredPath)
		}
	}
	return nil
}
