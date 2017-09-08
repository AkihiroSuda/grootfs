package store

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	errorspkg "github.com/pkg/errors"
)

func (s *StoreMeasurer) measurePath(path string) (int64, error) {
	stats := syscall.Statfs_t{}
	err := syscall.Statfs(path, &stats)
	if err != nil {
		return 0, errorspkg.Wrapf(err, "Invalid path %s", path)
	}

	bsize := uint64(stats.Bsize)
	free := stats.Bfree * bsize
	total := stats.Blocks * bsize

	return int64(total - free), nil
}

func (s *StoreMeasurer) measureCache(storePath string) (int64, error) {
	var cacheSize int64

	for _, volume := range s.volumeDriver.Volumes() {
		volumeSize, err := s.volumeDriver.VolumeSize(volume)
		if err != nil {
			return 0, err
		}
		cacheSize += volumeSize
	}

	for _, subdirectory := range []string{MetaDirName, TempDirName} {
		subdirSize, err := duUsage(filepath.Join(storePath, subdirectory))
		if err != nil {
			return 0, err
		}
		cacheSize += subdirSize
	}

	return cacheSize, nil
}

func duUsage(path string) (int64, error) {
	cmd := exec.Command("du", "-bs", path)
	stdoutBuffer := bytes.NewBuffer([]byte{})
	stderrBuffer := bytes.NewBuffer([]byte{})
	cmd.Stdout = stdoutBuffer
	cmd.Stderr = stdoutBuffer
	if err := cmd.Run(); err != nil {
		return 0, errorspkg.Wrapf(err, "du failed: %s", stderrBuffer.String())
	}

	usageString := strings.Split(stdoutBuffer.String(), "\t")[0]
	return strconv.ParseInt(usageString, 10, 64)
}
