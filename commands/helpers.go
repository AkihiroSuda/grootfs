package commands

import (
	"code.cloudfoundry.org/grootfs/commands/config"
	"code.cloudfoundry.org/grootfs/store/filesystems/btrfs"
	"code.cloudfoundry.org/grootfs/store/filesystems/overlayxfs"
	"code.cloudfoundry.org/grootfs/store/filesystems/storage"
	errorspkg "github.com/pkg/errors"
)

func createFileSystemDriver(cfg config.Config) (fileSystemDriver, error) {
	switch cfg.FSDriver {
	case "btrfs":
		return btrfs.NewDriver(cfg.BtrfsBin, cfg.DraxBin, cfg.StorePath)
	case "overlay-xfs":
		return overlayxfs.NewDriver(cfg.XFSProgsPath, cfg.StorePath)
	case "overlay-ext4":
		return storage.NewDriver("overlay-ext4", cfg.StorePath, cfg.UIDMappings, cfg.GIDMappings)
	case "new-overlay-xfs":
		return storage.NewDriver("new-overlay-xfs", cfg.StorePath, cfg.UIDMappings, cfg.GIDMappings)
	default:
		return nil, errorspkg.Errorf("filesystem driver not supported: %s", cfg.FSDriver)
	}
}
