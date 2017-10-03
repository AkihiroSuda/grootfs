// +build linux

package unpacker

import (
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

func createLink(targetPathFileDescriptor int, path string, symlinkName string) error {
	_path, err := syscall.BytePtrFromString(path)
	if err != nil {
		return errors.Wrap(err, "converting path to byte pointer")
	}

	_symlinkName, err := syscall.BytePtrFromString(symlinkName)
	if err != nil {
		return errors.Wrap(err, "converting link name to byte pointer")
	}

	_, _, errno := syscall.Syscall(
		syscall.SYS_LINKAT,
		uintptr(unsafe.Pointer(_path)),
		uintptr(targetPathFileDescriptor),
		uintptr(unsafe.Pointer(_symlinkName)),
	)

	if errno != 0 {
		return errno
	}

	return nil
}
