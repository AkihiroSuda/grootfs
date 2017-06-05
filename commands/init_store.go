package commands // import "code.cloudfoundry.org/grootfs/commands"

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/grootfs/commands/config"
	"code.cloudfoundry.org/grootfs/commands/idmapping_parser"
	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/grootfs/store/manager"
	"code.cloudfoundry.org/lager"

	errorspkg "github.com/pkg/errors"
	"github.com/urfave/cli"
)

var InitStoreCommand = cli.Command{
	Name:        "init-store",
	Usage:       "init-store --store <path>",
	Description: "Initialize a Store Directory",

	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "uid-mapping",
			Usage: "UID mapping for image translation, e.g.: <Namespace UID>:<Host UID>:<Size>",
		},
		cli.StringSliceFlag{
			Name:  "gid-mapping",
			Usage: "GID mapping for image translation, e.g.: <Namespace GID>:<Host GID>:<Size>",
		},
		cli.Int64Flag{
			Name:  "store-size-bytes",
			Usage: "Creates a new filesystem of the given size and mounts it to the given Store Directory",
		},
		cli.Int64Flag{
			Name:  "external-logdev-size-mb",
			Usage: "Size of the external log device when using XFS driver. 0 to disable. Only works with store-size-bytes option.",
			Value: 0,
		},
		cli.StringFlag{
			Name:  "rootless",
			Usage: "User and group of the store owner in the format <user>:<group>. If not supplied store will be owned by root.",
			Value: "",
		},
	},

	Action: func(ctx *cli.Context) error {
		logger := ctx.App.Metadata["logger"].(lager.Logger)
		logger = logger.Session("init-store")

		if ctx.NArg() != 0 {
			logger.Error("parsing-command", errorspkg.New("invalid arguments"), lager.Data{"args": ctx.Args()})
			return cli.NewExitError(fmt.Sprintf("invalid arguments - usage: %s", ctx.Command.Usage), 1)
		}

		configBuilder := ctx.App.Metadata["configBuilder"].(*config.Builder).
			WithStoreSizeBytes(ctx.Int64("store-size-bytes")).
			WithExternalLogdevSize(ctx.Int64("external-logdev-size-mb"))
		cfg, err := configBuilder.Build()
		logger.Debug("init-store", lager.Data{"currentConfig": cfg})
		if err != nil {
			logger.Error("config-builder-failed", err)
			return cli.NewExitError(err.Error(), 1)
		}

		storePath := cfg.StorePath

		if os.Getuid() != 0 {
			err := errorspkg.Errorf("store %s can only be initialized by Root user", storePath)
			logger.Error("init-store-failed", err)
			return cli.NewExitError(err.Error(), 1)
		}

		fsDriver, err := createFileSystemDriver(cfg)
		if err != nil {
			logger.Error("failed-to-initialise-driver", err)
			return cli.NewExitError(err.Error(), 1)
		}

		// uidMappings, gidMappings, err := mappingResolver.Resolve(ctx.String("rootless"), "uid-maping...")

		uidMappings, err := idmapping_parser.NewFlagMappingParser(ctx.StringSlice("uid-mapping")).ParseMappings()
		if err != nil {
			err = errorspkg.Errorf("parsing uid-mapping: %s", err)
			logger.Error("parsing-command", err)
			return cli.NewExitError(err.Error(), 1)
		}
		gidMappings, err := parseIDMappings(ctx.StringSlice("gid-mapping"))
		if err != nil {
			err = errorspkg.Errorf("parsing gid-mapping: %s", err)
			logger.Error("parsing-command", err)
			return cli.NewExitError(err.Error(), 1)
		}
		storeSizeBytes := cfg.Init.StoreSizeBytes

		namespacer := groot.NewStoreNamespacer(storePath)
		spec := manager.InitSpec{
			UIDMappings:    uidMappings,
			GIDMappings:    gidMappings,
			StoreSizeBytes: storeSizeBytes,
		}

		manager := manager.New(storePath, namespacer, fsDriver, fsDriver, fsDriver)
		if err := manager.InitStore(logger, spec); err != nil {
			logger.Error("cleaning-up-store-failed", err)
			return cli.NewExitError(errorspkg.Cause(err).Error(), 1)
		}

		return nil
	},
}
