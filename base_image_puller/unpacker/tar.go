package unpacker // import "code.cloudfoundry.org/grootfs/base_image_puller/unpacker"

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"

	"code.cloudfoundry.org/grootfs/base_image_puller"
	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/lager"
)

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

type defaultWhiteoutHandler struct{}

func (*defaultWhiteoutHandler) removeWhiteout(path string) error {
	toBeDeletedPath := strings.Replace(path, ".wh.", "", 1)
	if err := os.RemoveAll(toBeDeletedPath); err != nil {
		return errors.Wrap(err, "deleting whiteout file")
	}

	return nil
}

func (u *TarUnpacker) Unpack(logger lager.Logger, spec base_image_puller.UnpackSpec) (base_image_puller.UnpackOutput, error) {
	unpacker, err := NewTarUnpacker(u.strategy)
	if err != nil {
		return base_image_puller.UnpackOutput{}, errors.Wrap(err, "creating-tar-unpacker")
	}

	logger.Info("unpacking")
	var unpackOutput base_image_puller.UnpackOutput
	if unpackOutput, err = unpacker.unpack(logger, spec); err != nil {
		return base_image_puller.UnpackOutput{}, errors.Wrap(err, "unpacking-failed")
	}
	logger.Info("unpacking-completed")

	return unpackOutput, nil
}

func (u *TarUnpacker) unpack(logger lager.Logger, spec base_image_puller.UnpackSpec) (base_image_puller.UnpackOutput, error) {
	logger = logger.Session("unpacking-with-tar", lager.Data{"spec": spec})
	logger.Info("starting")
	defer logger.Info("ending")

	if err := safeMkdir(spec.TargetPath, 0755); err != nil {
		return base_image_puller.UnpackOutput{}, err
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	targetPathFileDescriptor, err := syscall.Open(spec.TargetPath, syscall.O_DIRECTORY, syscall.O_WRONLY)
	logger.Info(fmt.Sprintf("============== Unpacking at `%s`", spec.TargetPath))
	if err != nil {
		return base_image_puller.UnpackOutput{}, errors.Wrap(err, "failed to open target path directory")
	}

	tarReader := tar.NewReader(spec.Stream)
	opaqueWhiteouts := []string{}
	var totalBytesUnpacked int64
	for {
		tarHeader, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return base_image_puller.UnpackOutput{}, err
		}

		entryPath := filepath.Join(spec.BaseDirectory, tarHeader.Name)

		if strings.Contains(tarHeader.Name, ".wh..wh..opq") {
			opaqueWhiteouts = append(opaqueWhiteouts, entryPath)
			continue
		}

		if strings.Contains(tarHeader.Name, ".wh.") {
			if err = u.whiteoutHandler.removeWhiteout(entryPath); err != nil {
				return base_image_puller.UnpackOutput{}, err
			}
			continue
		}

		entrySize, err := u.handleEntry(logger, targetPathFileDescriptor, entryPath, tarReader, tarHeader, spec)
		if err != nil {
			return base_image_puller.UnpackOutput{}, err
		}

		totalBytesUnpacked += int64(entrySize)
	}

	return base_image_puller.UnpackOutput{
		BytesWritten:    totalBytesUnpacked,
		OpaqueWhiteouts: opaqueWhiteouts,
	}, nil
}

func (u *TarUnpacker) handleEntry(logger lager.Logger, targetPathFileDescriptor int, entryPath string, tarReader *tar.Reader, tarHeader *tar.Header, spec base_image_puller.UnpackSpec) (entrySize int, err error) {
	switch tarHeader.Typeflag {
	case tar.TypeBlock, tar.TypeChar:
		// ignore devices
		return 0, nil

	case tar.TypeLink:
		logger.Info(fmt.Sprintf("================ Creating link: %s\n%#v\n%#v", entryPath, tarHeader, spec))
		if err = u.createLink(targetPathFileDescriptor, entryPath, tarHeader); err != nil {
			return 0, err
		}

	case tar.TypeSymlink:
		logger.Info(fmt.Sprintf("================ Creating symlink: %s\n%#v\n%#v", entryPath, tarHeader, spec))
		if err = u.createSymlink(targetPathFileDescriptor, entryPath, tarHeader, spec); err != nil {
			return 0, err
		}

	case tar.TypeDir:
		logger.Info(fmt.Sprintf("================= Creating dir: %s\n%#v\n%#v", entryPath, tarHeader, spec))
		if err = u.createDirectory(targetPathFileDescriptor, entryPath, tarHeader, spec); err != nil {
			return 0, err
		}

	case tar.TypeReg, tar.TypeRegA:
		logger.Info(fmt.Sprintf("================== Creating file: %s\n%#v\n%#v", entryPath, tarHeader, spec))
		if entrySize, err = u.createRegularFile(targetPathFileDescriptor, entryPath, tarHeader, tarReader, spec); err != nil {
			return 0, err
		}
	}

	return entrySize, nil
}

func (u *TarUnpacker) createDirectory(targetPathFileDescriptor int, path string, tarHeader *tar.Header, spec base_image_puller.UnpackSpec) error {
	_, err := syscall.Openat(targetPathFileDescriptor, path, syscall.O_DIRECTORY, uint32(tarHeader.FileInfo().Mode()))
	if err != nil {
		err = syscall.Mkdirat(targetPathFileDescriptor, path, uint32(tarHeader.FileInfo().Mode()))
		if os.IsPermission(err) {
			dirName := filepath.Dir(tarHeader.Name)
			return errors.Errorf("'/%s' does not give write permission to its owner. This image can only be unpacked using uid and gid mappings, or by running as root.", dirName)
		}

		_, err = syscall.Openat(targetPathFileDescriptor, path, syscall.O_DIRECTORY, uint32(tarHeader.FileInfo().Mode()))
		if err != nil {
			return errors.Wrapf(err, "failed to open directory %s", path)
		}
	}

	if os.Getuid() == 0 {
		uid := u.translateID(tarHeader.Uid, spec.UIDMappings)
		gid := u.translateID(tarHeader.Gid, spec.GIDMappings)

		if err = syscall.Fchownat(targetPathFileDescriptor, path, uid, gid, 0); err != nil {
			return errors.Wrapf(err, "chowning directory %d:%d `%s`", uid, gid, path)
		}
	}

	// we need to explicitly apply perms because mkdir is subject to umask
	if err = syscall.Fchmodat(targetPathFileDescriptor, path, uint32(tarHeader.FileInfo().Mode()), 0); err != nil {
		return errors.Wrapf(err, "chmoding directory `%s`", path)
	}

	if err := changeModTime(targetPathFileDescriptor, path, tarHeader.ModTime); err != nil {
		return errors.Wrapf(err, "setting the modtime for directory `%s`", path)
	}

	return nil
}

func (u *TarUnpacker) createSymlink(targetPathFileDescriptor int, path string, tarHeader *tar.Header, spec base_image_puller.UnpackSpec) error {
	isSymlink, err := isSymlink(targetPathFileDescriptor, path)
	if err != nil {
		return errors.Wrapf(err, "determine whether `%s` is a symbolic link", path)
	}

	if isSymlink {
		if err = syscall.Unlinkat(targetPathFileDescriptor, path); err != nil {
			return errors.Wrapf(err, "removing symlink `%s`", path)
		}
	}

	if err = createSymLink(targetPathFileDescriptor, tarHeader.Linkname, path); err != nil {
		return errors.Wrapf(err, "create symlink `%s` -> `%s`", path, tarHeader.Linkname)
	}

	if err := changeModTime(targetPathFileDescriptor, path, tarHeader.ModTime); err != nil {
		return errors.Wrapf(err, "setting the modtime for the symlink `%s`", path)
	}

	if os.Getuid() == 0 {
		uid := u.translateID(tarHeader.Uid, spec.UIDMappings)
		gid := u.translateID(tarHeader.Gid, spec.GIDMappings)

		if err := syscall.Fchownat(targetPathFileDescriptor, path, uid, gid, 0); err != nil {
			return errors.Wrapf(err, "chowning link %d:%d `%s`", uid, gid, path)
		}
	}

	return nil
}

func (u *TarUnpacker) createLink(targetPathFileDescriptor int, path string, tarHeader *tar.Header) error {
	return createLink(targetPathFileDescriptor, path, tarHeader.Linkname)
}

func (u *TarUnpacker) createRegularFile(targetPathFileDescriptor int, path string, tarHeader *tar.Header, tarReader *tar.Reader, spec base_image_puller.UnpackSpec) (int, error) {
	fileDescriptor, err := syscall.Openat(targetPathFileDescriptor, path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, uint32(tarHeader.FileInfo().Mode()))
	if err != nil {
		newErr := errors.Wrapf(err, "creating file `%s`", path)

		if os.IsPermission(err) {
			dirName := filepath.Dir(tarHeader.Name)
			return 0, errors.Errorf("'/%s' does not give write permission to its owner. This image can only be unpacked using uid and gid mappings, or by running as root.", dirName)
		}

		return 0, newErr
	}

	tarContent, err := ioutil.ReadAll(tarReader)
	if err != nil {
		return 0, errors.Wrap(err, "reading tar")
	}

	fileSize, err := syscall.Write(fileDescriptor, tarContent)
	if err != nil {
		syscall.Close(fileDescriptor)
		return 0, errors.Wrapf(err, "writing to file `%s`", path)
	}

	if os.Getuid() == 0 {
		uid := u.translateID(tarHeader.Uid, spec.UIDMappings)
		gid := u.translateID(tarHeader.Gid, spec.GIDMappings)
		if err := syscall.Fchown(fileDescriptor, uid, gid); err != nil {
			return 0, errors.Wrapf(err, "chowning file %d:%d `%s`", uid, gid, path)
		}
	}

	// we need to explicitly apply perms because mkdir is subject to umask
	if err := syscall.Fchmod(fileDescriptor, uint32(tarHeader.FileInfo().Mode())); err != nil {
		return 0, errors.Wrapf(err, "chmoding file `%s`", path)
	}

	if err := changeModTime(targetPathFileDescriptor, path, tarHeader.ModTime); err != nil {
		return 0, errors.Wrapf(err, "setting the modtime for file `%s`", path)
	}

	if err := syscall.Close(fileDescriptor); err != nil {
		return 0, errors.Wrapf(err, "closing file `%s`", path)
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
