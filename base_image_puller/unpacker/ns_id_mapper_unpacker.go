package unpacker // import "code.cloudfoundry.org/grootfs/base_image_puller/unpacker"

import (
	"bytes"
	"encoding/json"
	"os"
	"syscall"

	"code.cloudfoundry.org/commandrunner"
	"github.com/containers/storage/pkg/reexec"
	"github.com/tscolari/lagregator"
	"github.com/urfave/cli"

	"code.cloudfoundry.org/grootfs/base_image_puller"
	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/lager"
	errorspkg "github.com/pkg/errors"
)

//go:generate counterfeiter . IDMapper

type IDMapper interface {
	MapUIDs(logger lager.Logger, pid int, mappings []groot.IDMappingSpec) error
	MapGIDs(logger lager.Logger, pid int, mappings []groot.IDMappingSpec) error
}

type NSIdMapperUnpacker struct {
	commandRunner  commandrunner.CommandRunner
	idMapper       IDMapper
	unpackStrategy UnpackStrategy
}

func init() {
	reexec.Register("unpack-wrapper", func() {
		cli.ErrWriter = os.Stdout
		logger := lager.NewLogger("unpack-wrapper")
		logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.DEBUG))

		if len(os.Args) != 3 {
			logger.Error("parsing-command", errorspkg.New("destination directory or filesystem were not specified"))
			os.Exit(1)
		}

		ctrlPipeR := os.NewFile(3, "/ctrl/pipe")
		buffer := make([]byte, 1)
		logger.Debug("waiting-for-control-pipe")
		_, err := ctrlPipeR.Read(buffer)
		if err != nil {
			logger.Error("reading-control-pipe", err)
			os.Exit(1)
		}
		logger.Debug("got-back-from-control-pipe")

		// Once all id mappings are set, we need to spawn the untar function
		// in a child proccess, so it can make use of it
		targetDir := os.Args[1]
		unpackStrategyJson := os.Args[2]
		cmd := reexec.Command("unpack", targetDir, unpackStrategyJson)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		logger.Debug("starting-unpack", lager.Data{
			"path": cmd.Path,
			"args": cmd.Args,
		})
		if err := cmd.Run(); err != nil {
			logger.Error("unpack-command-failed", err)
			os.Exit(1)
		}
		logger.Debug("unpack-command-done")
	})
}

func NewNSIdMapperUnpacker(commandRunner commandrunner.CommandRunner, idMapper IDMapper, strategy UnpackStrategy) *NSIdMapperUnpacker {
	return &NSIdMapperUnpacker{
		commandRunner:  commandRunner,
		idMapper:       idMapper,
		unpackStrategy: strategy,
	}
}

func (u *NSIdMapperUnpacker) Unpack(logger lager.Logger, spec base_image_puller.UnpackSpec) error {
	logger = logger.Session("ns-id-mapper-unpacking", lager.Data{"spec": spec})
	logger.Debug("starting")
	defer logger.Debug("ending")

	ctrlPipeR, ctrlPipeW, err := os.Pipe()
	if err != nil {
		return errorspkg.Wrap(err, "creating tar control pipe")
	}

	unpackStrategyJSON, err := json.Marshal(&u.unpackStrategy)
	if err != nil {
		logger.Error("unmarshal-unpack-strategy-failed", err)
		return errorspkg.Wrap(err, "unmarshal unpack strategy")
	}

	unpackCmd := reexec.Command("unpack-wrapper", spec.TargetPath, spec.BaseDirectory, string(unpackStrategyJSON))
	unpackCmd.Stdin = spec.Stream
	if len(spec.UIDMappings) > 0 || len(spec.GIDMappings) > 0 {
		unpackCmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUSER,
		}
	}

	outBuffer := bytes.NewBuffer([]byte{})
	unpackCmd.Stdout = outBuffer
	unpackCmd.Stderr = lagregator.NewRelogger(logger)
	unpackCmd.ExtraFiles = []*os.File{ctrlPipeR}

	logger.Debug("starting-unpack-wrapper-command", lager.Data{
		"path": unpackCmd.Path,
		"args": unpackCmd.Args,
	})
	if err := u.commandRunner.Start(unpackCmd); err != nil {
		return errorspkg.Wrap(err, "starting unpack command")
	}
	logger.Debug("unpack-wrapper-command-is-started")

	if err := u.setIDMappings(logger, spec, unpackCmd.Process.Pid); err != nil {
		_ = ctrlPipeW.Close()
		return err
	}

	if _, err := ctrlPipeW.Write([]byte{0}); err != nil {
		return errorspkg.Wrap(err, "writing to tar control pipe")
	}
	logger.Debug("unpack-wrapper-command-is-signaled-to-continue")

	logger.Debug("waiting-for-unpack-wrapper-command")
	if err := u.commandRunner.Wait(unpackCmd); err != nil {
		return errorspkg.Errorf(outBuffer.String())
	}
	logger.Debug("unpack-wrapper-command-done")

	return nil
}

func (u *NSIdMapperUnpacker) setIDMappings(logger lager.Logger, spec base_image_puller.UnpackSpec, untarPid int) error {
	if len(spec.UIDMappings) > 0 {
		if err := u.idMapper.MapUIDs(logger, untarPid, spec.UIDMappings); err != nil {
			return errorspkg.Wrap(err, "setting uid mapping")
		}
		logger.Debug("uid-mappings-are-set")
	}

	if len(spec.GIDMappings) > 0 {
		if err := u.idMapper.MapGIDs(logger, untarPid, spec.GIDMappings); err != nil {
			return errorspkg.Wrap(err, "setting gid mapping")
		}
		logger.Debug("gid-mappings-are-set")
	}

	return nil
}
