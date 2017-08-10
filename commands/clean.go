package commands // import "code.cloudfoundry.org/grootfs/commands"

import (
	"fmt"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/grootfs/commands/config"
	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/grootfs/metrics"
	storepkg "code.cloudfoundry.org/grootfs/store"
	"code.cloudfoundry.org/grootfs/store/dependency_manager"
	"code.cloudfoundry.org/grootfs/store/garbage_collector"
	imageClonerpkg "code.cloudfoundry.org/grootfs/store/image_cloner"
	locksmithpkg "code.cloudfoundry.org/grootfs/store/locksmith"
	errorspkg "github.com/pkg/errors"

	"github.com/urfave/cli"
)

var CleanCommand = cli.Command{
	Name:        "clean",
	Usage:       "clean",
	Description: "Cleans up unused layers",

	Flags: []cli.Flag{
		cli.Int64Flag{
			Name:  "threshold-bytes",
			Usage: "Disk usage of the store directory at which cleanup should trigger.",
		},
		cli.StringSliceFlag{
			Name:  "ignore-image",
			Usage: "Images to ignore during cleanup",
		},
	},

	Action: func(ctx *cli.Context) error {
		logger := ctx.App.Metadata["logger"].(lager.Logger)
		logger = logger.Session("clean")
		newExitError := newErrorHandler(logger, "clean")

		configBuilder := ctx.App.Metadata["configBuilder"].(*config.Builder)
		configBuilder.WithIgnoreBaseImages(ctx.StringSlice("ignore-image"))
		configBuilder.WithCleanThresholdBytes(ctx.Int64("threshold-bytes"),
			ctx.IsSet("threshold-bytes"))

		cfg, err := configBuilder.Build()
		logger.Debug("clean-config", lager.Data{"currentConfig": cfg})
		if err != nil {
			logger.Error("config-builder-failed", err)
			return newExitError(err.Error(), 1)
		}

		storePath := cfg.StorePath
		if _, err := os.Stat(storePath); os.IsNotExist(err) {
			err := errorspkg.Errorf("no store found at %s", storePath)
			logger.Error("store-path-failed", err, nil)
			return newExitError(err.Error(), 0)
		}

		fsDriver, err := createFileSystemDriver(cfg)
		if err != nil {
			logger.Error("failed-to-initialise-driver", err)
			return newExitError(err.Error(), 1)
		}

		imageCloner := imageClonerpkg.NewImageCloner(fsDriver, storePath)
		metricsEmitter := metrics.NewEmitter(systemReporter(cfg.SlowActionThreshold))

		locksmith := locksmithpkg.NewExclusiveFileSystem(storePath, metricsEmitter)
		dependencyManager := dependency_manager.NewDependencyManager(
			filepath.Join(storePath, storepkg.MetaDirName, "dependencies"),
		)
		sm := storepkg.NewStoreMeasurer(storePath)
		gc := garbage_collector.NewGC(fsDriver, imageCloner, dependencyManager)

		cleaner := groot.IamCleaner(locksmith, sm, gc, metricsEmitter)

		noop, err := cleaner.Clean(logger, cfg.Clean.ThresholdBytes, cfg.Clean.IgnoreBaseImages)
		if err != nil {
			logger.Error("cleaning-up-unused-resources", err)
			return newExitError(err.Error(), 1)
		}

		if noop {
			fmt.Println("threshold not reached: skipping clean")
			return nil
		}

		fmt.Println("clean completed")

		usage, err := sm.MeasureStore(logger)
		if err != nil {
			logger.Error("measuring-store", err)
			return newExitError(err.Error(), 1)
		}

		metricsEmitter.TryIncrementRunCount("clean", nil)
		metricsEmitter.TryEmitUsage(logger, "StoreUsage", usage)
		return nil
	},
}
