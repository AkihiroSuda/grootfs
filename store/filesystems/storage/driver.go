package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/grootfs/store"
	"code.cloudfoundry.org/grootfs/store/image_cloner"
	"code.cloudfoundry.org/lager"
	"github.com/containers/storage/drivers"
	"github.com/containers/storage/drivers/overlay2"
	"github.com/containers/storage/pkg/idtools"
	errorspkg "github.com/pkg/errors"
)

func NewDriver(filesystem, storePath string, uidMaps, gidMaps []string) (*Driver, error) {
	storageUIDMap, err := parseIDMappings(uidMaps)
	if err != nil {
		return nil, errorspkg.Wrap(err, "Failed to parse UID mappings")
	}
	storageGIDMap, err := parseIDMappings(gidMaps)
	if err != nil {
		return nil, errorspkg.Wrap(err, "Failed to parse GID mappings")
	}

	var driver graphdriver.Driver

	switch filesystem {
	case "overlay-ext4":
		driver, err = overlay2.Init(storePath, []string{}, storageUIDMap, storageGIDMap)
	default:
		err = errorspkg.Errorf("Unsupported filesystem: %s", filesystem)
	}

	return &Driver{
		storageDriver: driver,
		storePath:     storePath,
	}, err
}

type Driver struct {
	storageDriver graphdriver.Driver
	storePath     string
}

func (d *Driver) VolumePath(logger lager.Logger, id string) (string, error) {
	volPath := filepath.Join(d.storePath, id)
	_, err := os.Stat(volPath)
	if err == nil {
		return volPath, nil
	}

	return "", errorspkg.Wrapf(err, "volume does not exist `%s`", id)

	return volPath, nil
}

func (d *Driver) CreateVolume(logger lager.Logger, parentID, id string) (string, error) {
	volumePath := filepath.Join(d.storePath, id)
	err := d.storageDriver.Create(id, parentID, "", map[string]string{})
	if err != nil {
		return "", err
	}
	return volumePath, err
}

func (d *Driver) DestroyVolume(logger lager.Logger, id string) error {
	// remove symlinked volumes
	volumePath := filepath.Join(d.storePath, "volumes", id)
	if err := os.RemoveAll(volumePath); err != nil {
		logger.Error(fmt.Sprintf("failed to destroy volume %s", volumePath), err)
		return errorspkg.Wrapf(err, "destroying volume (%s)", id)
	}

	// remove actual volumes
	if err := d.storageDriver.Remove(id); err != nil {
		return errorspkg.Wrap(err, "failed to remove volume related stuffz")
	}

	return nil
}

func (d *Driver) MoveVolume(from, to string) error {
	var err error
	if err = os.Rename(from, to); err != nil {
		return errorspkg.Wrap(err, "moving volume")
	}

	linkFileContents, err := ioutil.ReadFile(filepath.Join(to, "link"))
	if err != nil {
		return errorspkg.Wrap(err, "failed to read volume link file")
	}

	linkPath := path.Join(d.storePath, "l", string(linkFileContents))
	if err := os.Remove(linkPath); err != nil {
		return errorspkg.Wrap(err, "unable to delete stale symlink")
	}

	// path.Join should really end with 'diff' directory, but this means we need
	// to change the directory we unpack to
	volumeId := filepath.Base(to)
	if err := os.Symlink(path.Join("..", volumeId), linkPath); err != nil {
		return err
	}

	if err := os.Symlink(to, filepath.Join(d.storePath, store.VolumesDirName, volumeId)); err != nil {
		return err
	}

	return nil
}

func (d *Driver) Volumes(logger lager.Logger) ([]string, error) {
	volumes := []string{}
	existingVolumes, err := ioutil.ReadDir(path.Join(d.storePath, store.VolumesDirName))
	if err != nil {
		return nil, errorspkg.Wrap(err, "failed to list volumes")
	}

	for _, volumeInfo := range existingVolumes {
		volumes = append(volumes, volumeInfo.Name())
	}
	return volumes, nil
}

func (d *Driver) CreateImage(logger lager.Logger, spec image_cloner.ImageDriverSpec) error {
	id := filepath.Base(spec.ImagePath)
	destination := path.Join(d.storePath, "images", id)
	os.Mkdir(destination, 755)

	if err := d.storageDriver.CreateReadWrite(id, spec.BaseVolumeIDs[0], "", map[string]string{}); err != nil {
		return errorspkg.Wrap(err, "failed to create read/write layer")
	}

	source := filepath.Join(d.storePath, id, "merged")
	if err := os.Symlink(source, filepath.Join(destination, "rootfs")); err != nil {
		return err
	}

	_, err := d.storageDriver.Get(id, "")
	if err != nil {
		return errorspkg.Wrap(err, "failed to create and mount fs")
	}

	return nil
}

func (d *Driver) DestroyImage(logger lager.Logger, path string) error {
	id := filepath.Base(path)

	if err := d.storageDriver.Put(id); err != nil {
		return errorspkg.Wrap(err, "failed to unmount image fs")
	}

	if err := d.storageDriver.Remove(id); err != nil {
		return errorspkg.Wrap(err, "failed to remove image related stuffz")
	}

	return nil
}

func (d *Driver) FetchStats(logger lager.Logger, path string) (groot.VolumeStats, error) {
	return groot.VolumeStats{}, nil
}

func parseIDMappings(args []string) ([]idtools.IDMap, error) {
	if len(args) == 0 {
		return nil, nil
	}

	mappings := []idtools.IDMap{}

	for _, v := range args {
		var mapping idtools.IDMap
		_, err := fmt.Sscanf(v, "%d:%d:%d", &mapping.ContainerID, &mapping.HostID, &mapping.Size)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
	}

	return mappings, nil
}
