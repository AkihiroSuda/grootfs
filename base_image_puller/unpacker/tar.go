package unpacker // import "code.cloudfoundry.org/grootfs/base_image_puller/unpacker"

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/tscolari/lagregator"

	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/system"
	"github.com/urfave/cli"

	"code.cloudfoundry.org/grootfs/base_image_puller"
	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/lager"
)

func init() {
	var fail = func(logger lager.Logger, message string, err error) {
		logger.Error(message, err)
		fmt.Println(err.Error())
		os.Exit(1)
	}

	reexec.Register("unpack", func() {
		cli.ErrWriter = os.Stdout
		logger := lager.NewLogger("unpack")
		logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.DEBUG))

		rootFSPath := os.Args[1]
		unpackStrategyJSON := os.Args[2]

		var unpackStrategy UnpackStrategy
		if err := json.Unmarshal([]byte(unpackStrategyJSON), &unpackStrategy); err != nil {
			fail(logger, "unmarshal-unpack-strategy-failed", err)
		}

		unpacker, err := NewTarUnpacker(unpackStrategy)
		if err != nil {
			fail(logger, "creating-tar-unpacker", err)
		}

		var totalUnpacked int64
		if totalUnpacked, err = unpacker.Unpack(logger, base_image_puller.UnpackSpec{
			Stream:     os.Stdin,
			TargetPath: rootFSPath,
		}); err != nil {
			fail(logger, "unpacking-failed", err)
		}

		fmt.Fprintf(os.Stdout, "%d", totalUnpacked)
	})

	reexec.Register("chroot-unpack", func() {
		cli.ErrWriter = os.Stdout
		logger := lager.NewLogger("chroot")
		logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.DEBUG))

		unpackSpecJSON := os.Args[1]
		unpackStrategyJSON := os.Args[2]

		var unpackSpec base_image_puller.UnpackSpec
		if err := json.Unmarshal([]byte(unpackSpecJSON), &unpackSpec); err != nil {
			fail(logger, "unmarshal-unpack-spec-failed", err)
		}

		var unpackStrategy UnpackStrategy
		if err := json.Unmarshal([]byte(unpackStrategyJSON), &unpackStrategy); err != nil {
			fail(logger, "unmarshal-unpack-strategy-failed", err)
		}

		unpacker, err := NewTarUnpacker(unpackStrategy)
		if err != nil {
			fail(logger, "creating-tar-unpacker", err)
		}

		unpackSpec.Stream = os.Stdin

		var totalUnpacked int64
		if totalUnpacked, err = unpacker.unpack(logger, unpackSpec); err != nil {
			fail(logger, "unpacking-failed", err)
		}

		fmt.Fprintf(os.Stdout, "%d", totalUnpacked)
	})
}

type UnpackStrategy struct {
	Name               string
	WhiteoutDevicePath string
}

type TarUnpacker struct {
	whiteoutHandler whiteoutHandler
	strategy        UnpackStrategy
}

func NewTarUnpacker(unpackStrategy UnpackStrategy) (*TarUnpacker, error) {
	var woHandler whiteoutHandler

	switch unpackStrategy.Name {
	case "overlay-xfs":
		parentDirectory := filepath.Dir(unpackStrategy.WhiteoutDevicePath)
		whiteoutDevDir, err := os.Open(parentDirectory)
		if err != nil {
			return nil, err
		}

		woHandler = &overlayWhiteoutHandler{
			whiteoutDevName: filepath.Base(unpackStrategy.WhiteoutDevicePath),
			whiteoutDevDir:  whiteoutDevDir,
		}
	default:
		woHandler = &defaultWhiteoutHandler{}
	}

	return &TarUnpacker{
		whiteoutHandler: woHandler,
		strategy:        unpackStrategy,
	}, nil
}

type whiteoutHandler interface {
	removeWhiteout(path string) error
	removeOpaqueWhiteouts(paths []string) error
}

type overlayWhiteoutHandler struct {
	whiteoutDevName string
	whiteoutDevDir  *os.File
}

func (h *overlayWhiteoutHandler) removeWhiteout(path string) error {
	toBeDeletedPath := strings.Replace(path, ".wh.", "", 1)
	if err := os.RemoveAll(toBeDeletedPath); err != nil {
		return errors.Wrap(err, "deleting  file")
	}

	targetPath, err := os.Open(filepath.Dir(toBeDeletedPath))
	if err != nil {
		return errors.Wrap(err, "opening target whiteout directory")
	}

	targetName, err := syscall.BytePtrFromString(filepath.Base(toBeDeletedPath))
	if err != nil {
		return errors.Wrap(err, "converting whiteout path to byte pointer")
	}

	whiteoutDevName, err := syscall.BytePtrFromString(h.whiteoutDevName)
	if err != nil {
		return errors.Wrap(err, "converting whiteout device name to byte pointer")
	}

	_, _, errno := syscall.Syscall6(syscall.SYS_LINKAT,
		h.whiteoutDevDir.Fd(),
		uintptr(unsafe.Pointer(whiteoutDevName)),
		targetPath.Fd(),
		uintptr(unsafe.Pointer(targetName)),
		0,
		0,
	)

	if errno != 0 {
		return errors.Wrapf(errno, "failed to create whiteout node: %s", toBeDeletedPath)
	}

	return nil
}

func (*overlayWhiteoutHandler) removeOpaqueWhiteouts(paths []string) error {
	for _, path := range paths {
		parentDir := filepath.Dir(path)
		if err := system.Lsetxattr(parentDir, "trusted.overlay.opaque", []byte("y"), 0); err != nil {
			return errors.Wrapf(err, "set xattr for %s", parentDir)
		}

		if err := cleanWhiteoutDir(parentDir); err != nil {
			return errors.Wrapf(err, "clean without dir %s", parentDir)
		}
	}

	return nil
}

type defaultWhiteoutHandler struct{}

func (*defaultWhiteoutHandler) removeWhiteout(path string) error {
	toBeDeletedPath := strings.Replace(path, ".wh.", "", 1)
	if err := os.RemoveAll(toBeDeletedPath); err != nil {
		return errors.Wrap(err, "deleting whiteout file")
	}

	return nil
}

func (*defaultWhiteoutHandler) removeOpaqueWhiteouts(paths []string) error {
	for _, p := range paths {
		parentDir := path.Dir(p)
		if err := cleanWhiteoutDir(parentDir); err != nil {
			return err
		}
	}

	return nil
}

func (u *TarUnpacker) Unpack(logger lager.Logger, spec base_image_puller.UnpackSpec) (int64, error) {
	strategyJSON, err := json.Marshal(u.strategy)
	if err != nil {
		return 0, err
	}

	unpackSpecJSON, err := json.Marshal(spec)
	if err != nil {
		return 0, err
	}

	cmd := reexec.Command("chroot-unpack", string(unpackSpecJSON), string(strategyJSON))
	cmd.Stderr = lagregator.NewRelogger(logger)
	cmd.Stdin = spec.Stream

	output, err := cmd.Output()
	if err != nil {
		logger.Error("chroot unpack failed", err, lager.Data{"output": string(output)})
		return 0, errors.New(strings.TrimSpace(string(output)))
	}

	totalUnpacked, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		logger.Error("unpack-invalid-output", err, lager.Data{"output": string(output)})
		return 0, errors.Wrap(err, "parsing unpack output")
	}
	return totalUnpacked, nil
}

func (u *TarUnpacker) unpack(logger lager.Logger, spec base_image_puller.UnpackSpec) (int64, error) {
	logger = logger.Session("unpacking-with-tar", lager.Data{"spec": spec})
	logger.Info("starting")
	defer logger.Info("ending")

	if err := safeMkdir(spec.TargetPath, 0755); err != nil {
		return 0, err
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := chroot(spec.TargetPath); err != nil {
		return 0, errors.Wrap(err, "failed to chroot")
	}

	tarReader := tar.NewReader(spec.Stream)
	opaqueWhiteouts := []string{}
	var totalBytesUnpacked int64
	for {
		tarHeader, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return 0, err
		}

		if strings.Contains(tarHeader.Name, ".wh..wh..opq") {
			opaqueWhiteouts = append(opaqueWhiteouts, tarHeader.Name)
			continue
		}
		if strings.Contains(tarHeader.Name, ".wh.") {
			if err := u.whiteoutHandler.removeWhiteout(tarHeader.Name); err != nil {
				return 0, err
			}
			continue
		}

		entrySize, err := u.handleEntry(tarHeader.Name, tarReader, tarHeader, spec)
		if err != nil {
			return 0, err
		}

		totalBytesUnpacked += entrySize
	}

	return totalBytesUnpacked, u.whiteoutHandler.removeOpaqueWhiteouts(opaqueWhiteouts)
}

func (u *TarUnpacker) handleEntry(entryPath string, tarReader *tar.Reader, tarHeader *tar.Header, spec base_image_puller.UnpackSpec) (entrySize int64, err error) {
	switch tarHeader.Typeflag {
	case tar.TypeBlock, tar.TypeChar:
		// ignore devices
		return 0, nil

	case tar.TypeLink:
		if err = u.createLink(entryPath, tarHeader); err != nil {
			return 0, err
		}

	case tar.TypeSymlink:
		if err = u.createSymlink(entryPath, tarHeader, spec); err != nil {
			return 0, err
		}

	case tar.TypeDir:
		if err = u.createDirectory(entryPath, tarHeader, spec); err != nil {
			return 0, err
		}

	case tar.TypeReg, tar.TypeRegA:
		if entrySize, err = u.createRegularFile(entryPath, tarHeader, tarReader, spec); err != nil {
			return 0, err
		}
	}

	return entrySize, nil
}

func (u *TarUnpacker) createDirectory(path string, tarHeader *tar.Header, spec base_image_puller.UnpackSpec) error {
	if _, err := os.Stat(path); err != nil {
		if err = os.Mkdir(path, tarHeader.FileInfo().Mode()); err != nil {
			newErr := errors.Wrapf(err, "creating directory `%s`", path)

			if os.IsPermission(err) {
				dirName := filepath.Dir(tarHeader.Name)
				return errors.Errorf("'/%s' does not give write permission to its owner. This image can only be unpacked using uid and gid mappings, or by running as root.", dirName)
			}

			return newErr
		}
	}

	if os.Getuid() == 0 {
		uid := u.translateID(tarHeader.Uid, spec.UIDMappings)
		gid := u.translateID(tarHeader.Gid, spec.GIDMappings)
		if err := os.Chown(path, uid, gid); err != nil {
			return errors.Wrapf(err, "chowning directory %d:%d `%s`", uid, gid, path)
		}
	}

	// we need to explicitly apply perms because mkdir is subject to umask
	if err := os.Chmod(path, tarHeader.FileInfo().Mode()); err != nil {
		return errors.Wrapf(err, "chmoding directory `%s`", path)
	}

	if err := changeModTime(path, tarHeader.ModTime); err != nil {
		return errors.Wrapf(err, "setting the modtime for directory `%s`: %s", path)
	}

	return nil
}

func (u *TarUnpacker) createSymlink(path string, tarHeader *tar.Header, spec base_image_puller.UnpackSpec) error {
	if _, err := os.Lstat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return errors.Wrapf(err, "removing file `%s`", path)
		}
	}

	if err := os.Symlink(tarHeader.Linkname, path); err != nil {
		return errors.Wrapf(err, "create symlink `%s` -> `%s`", tarHeader.Linkname, path)
	}

	if err := changeModTime(path, tarHeader.ModTime); err != nil {
		return errors.Wrapf(err, "setting the modtime for the symlink `%s`", path)
	}

	if os.Getuid() == 0 {
		uid := u.translateID(tarHeader.Uid, spec.UIDMappings)
		gid := u.translateID(tarHeader.Gid, spec.GIDMappings)

		if err := os.Lchown(path, uid, gid); err != nil {
			return errors.Wrapf(err, "chowning link %d:%d `%s`", uid, gid, path)
		}
	}

	return nil
}

func (u *TarUnpacker) createLink(path string, tarHeader *tar.Header) error {
	return os.Link(tarHeader.Linkname, path)
}

func (u *TarUnpacker) createRegularFile(path string, tarHeader *tar.Header, tarReader *tar.Reader, spec base_image_puller.UnpackSpec) (int64, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, tarHeader.FileInfo().Mode())
	if err != nil {
		newErr := errors.Wrapf(err, "creating file `%s`", path)

		if os.IsPermission(err) {
			dirName := filepath.Dir(tarHeader.Name)
			return 0, errors.Errorf("'/%s' does not give write permission to its owner. This image can only be unpacked using uid and gid mappings, or by running as root.", dirName)
		}

		return 0, newErr
	}

	fileSize, err := io.Copy(file, tarReader)
	if err != nil {
		_ = file.Close()
		return 0, errors.Wrapf(err, "writing to file `%s`", path)
	}

	if err := file.Close(); err != nil {
		return 0, errors.Wrapf(err, "closing file `%s`", path)
	}

	if os.Getuid() == 0 {
		uid := u.translateID(tarHeader.Uid, spec.UIDMappings)
		gid := u.translateID(tarHeader.Gid, spec.GIDMappings)
		if err := os.Chown(path, uid, gid); err != nil {
			return 0, errors.Wrapf(err, "chowning file %d:%d `%s`", uid, gid, path)
		}
	}

	// we need to explicitly apply perms because mkdir is subject to umask
	if err := os.Chmod(path, tarHeader.FileInfo().Mode()); err != nil {
		return 0, errors.Wrapf(err, "chmoding file `%s`", path)
	}

	if err := changeModTime(path, tarHeader.ModTime); err != nil {
		return 0, errors.Wrapf(err, "setting the modtime for file `%s`", path)
	}

	return fileSize, nil
}

func cleanWhiteoutDir(path string) error {
	contents, err := ioutil.ReadDir(path)
	if err != nil {
		return errors.Wrap(err, "reading whiteout directory")
	}

	for _, content := range contents {
		if err := os.RemoveAll(filepath.Join(path, content.Name())); err != nil {
			return errors.Wrap(err, "cleaning up whiteout directory")
		}
	}

	return nil
}

func (u *TarUnpacker) translateID(id int, mappings []groot.IDMappingSpec) int {
	if id == 0 {
		return u.translateRootID(mappings)
	}

	for _, mapping := range mappings {
		if mapping.Size == 1 {
			continue
		}

		if id >= mapping.NamespaceID && id < mapping.NamespaceID+mapping.Size {
			return mapping.HostID + id - 1
		}
	}

	return id
}

func (u *TarUnpacker) translateRootID(mappings []groot.IDMappingSpec) int {
	for _, mapping := range mappings {
		if mapping.Size == 1 {
			return mapping.HostID
		}
	}

	return 0
}

func safeMkdir(path string, perm os.FileMode) error {
	if _, err := os.Stat(path); err != nil {
		if err := os.Mkdir(path, perm); err != nil {
			return errors.Wrapf(err, "making destination directory `%s`", path)
		}
	}
	return nil
}

func chroot(path string) error {
	if err := syscall.Chroot(path); err != nil {
		return err
	}

	if err := os.Chdir("/"); err != nil {
		return err
	}

	return nil
}
