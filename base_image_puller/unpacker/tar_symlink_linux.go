// +build linux

package unpacker

import (
	"bytes"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

func createSymLink(targetPathFileDescriptor int, path string, symlinkName string) error {
	_path, err := syscall.BytePtrFromString(path)
	if err != nil {
		return errors.Wrap(err, "converting path to byte pointer")
	}

	_symlinkName, err := syscall.BytePtrFromString(symlinkName)
	if err != nil {
		return errors.Wrap(err, "converting symlink name to byte pointer")
	}

	_, _, errno := syscall.Syscall(
		syscall.SYS_SYMLINKAT,
		uintptr(unsafe.Pointer(_path)),
		uintptr(targetPathFileDescriptor),
		uintptr(unsafe.Pointer(_symlinkName)),
	)

	if errno != 0 {
		return errno
	}

	return nil
}

func isSymlink(targetPathFileDescriptor int, symlinkName string) (bool, error) {

	_symlinkName, err := syscall.BytePtrFromString(symlinkName)
	if err != nil {
		return false, errors.Wrap(err, "converting symlink name to byte pointer")
	}

	outputBuffer := bytes.NewBufferString("")
	_buffer, err := syscall.BytePtrFromString(outputBuffer.String())
	if err != nil {
		return false, errors.Wrap(err, "converting empty string to byte pointer")
	}

	_, _, errno := syscall.Syscall6(
		syscall.SYS_READLINKAT,
		uintptr(targetPathFileDescriptor),
		uintptr(unsafe.Pointer(_symlinkName)),
		uintptr(unsafe.Pointer(_buffer)),
		uintptr(outputBuffer.Len()),
		0, 0,
	)

	switch errno {
	case 0:
		return true, nil
	case syscall.ENOENT:
		return false, nil
	case syscall.EINVAL:
		return false, nil
	default:
		return false, err
	}
}
