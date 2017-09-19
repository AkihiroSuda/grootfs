package commands // import "code.cloudfoundry.org/grootfs/commands"

import (
	"fmt"
	"path/filepath"

	"code.cloudfoundry.org/commandrunner/linux_command_runner"
	unpackerpkg "code.cloudfoundry.org/grootfs/base_image_puller/unpacker"
	"code.cloudfoundry.org/grootfs/commands/config"
	"code.cloudfoundry.org/grootfs/commands/idfinder"
	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/grootfs/metrics"
	"code.cloudfoundry.org/grootfs/store"
	"code.cloudfoundry.org/grootfs/store/dependency_manager"
	"code.cloudfoundry.org/grootfs/store/filesystems/namespaced"
	"code.cloudfoundry.org/grootfs/store/image_cloner"
	"code.cloudfoundry.org/lager"
	errorspkg "github.com/pkg/errors"
	"github.com/urfave/cli"
)

var DeleteCommand = cli.Command{
	Name:        "delete",
	Usage:       "delete <id|image path>",
	Description: "Deletes a container image",

	Action: func(ctx *cli.Context) error {
		logger := ctx.App.Metadata["logger"].(lager.Logger)
		logger = logger.Session("delete")
		newExitError := newErrorHandler(logger, "delete")

		if ctx.NArg() != 1 {
			logger.Error("parsing-command", errorspkg.New("id was not specified"))
			return newExitError("id was not specified", 1)
		}

		configBuilder := ctx.App.Metadata["configBuilder"].(*config.Builder)
		cfg, err := configBuilder.Build()
		logger.Debug("delete-config", lager.Data{"currentConfig": cfg})
		if err != nil {
			logger.Error("config-builder-failed", err)
			return newExitError(err.Error(), 1)
		}

		storePath := cfg.StorePath
		idOrPath := ctx.Args().First()
		id, err := idfinder.FindID(storePath, idOrPath)
		if err != nil {
			logger.Debug("id-not-found-skipping", lager.Data{"id": idOrPath, "storePath": storePath, "errorMessage": err.Error()})
			fmt.Println(err)
			return nil
		}

		fsDriver, err := createFileSystemDriver(cfg)
		if err != nil {
			logger.Error("failed-to-initialise-driver", err)
			return newExitError(err.Error(), 1)
		}

		storeNamespacer := groot.NewStoreNamespacer(storePath)
		idMappings, _ := storeNamespacer.Read()
		runner := linux_command_runner.New()
		idMapper := unpackerpkg.NewIDMapper(cfg.NewuidmapBin, cfg.NewgidmapBin, runner)
		nsFsDriver := namespaced.New(fsDriver, idMappings, idMapper, runner)

		imageCloner := image_cloner.NewImageCloner(nsFsDriver, storePath)
		dependencyManager := dependency_manager.NewDependencyManager(
			filepath.Join(storePath, store.MetaDirName, "dependencies"),
		)
		metricsEmitter := metrics.NewEmitter()
		deleter := groot.IamDeleter(imageCloner, dependencyManager, metricsEmitter)

		err = deleter.Delete(logger, id)
		if err != nil {
			logger.Error("deleting-image-failed", err)
			return newExitError(err.Error(), 1)
		}

		fmt.Printf("Image %s deleted\n", id)
		metricsEmitter.TryIncrementRunCount("delete", nil)
		return nil
	},
}
