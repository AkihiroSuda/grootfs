package testhelpers

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"code.cloudfoundry.org/grootfs/store"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

func CleanUpExternalLogDevice(externalLogPath string) {
	externalLogPathLoopDevice := bytes.NewBuffer([]byte{})
	getExternalLogDeviceCmd := exec.Command("sh", "-c", fmt.Sprintf("losetup -a | grep %s.external-log | cut -d : -f 1", externalLogPath))
	getExternalLogDeviceCmd.Stdout = io.MultiWriter(GinkgoWriter, externalLogPathLoopDevice)
	getExternalLogDeviceCmd.Stderr = GinkgoWriter
	Expect(getExternalLogDeviceCmd.Run()).To(Succeed())

	cleanExternalLogDeviceCmd := exec.Command("sh", "-c", fmt.Sprintf("losetup -d %s", externalLogPathLoopDevice.String()))
	cleanExternalLogDeviceCmd.Stdout = GinkgoWriter
	cleanExternalLogDeviceCmd.Stderr = GinkgoWriter
	Expect(cleanExternalLogDeviceCmd.Run()).To(Succeed())
}

func CleanUpOverlayMounts(mountPath string) {
	var mountPoints []string

	Expect(exec.Command("/bin/bash", "-c", "for m in $(cat /proc/mounts | grep overlay | cut -d ' ' -f 2); do umount $m; done").Run()).To(Succeed())

	for i := 0; i < 10; i++ {
		mountPoints = internalMountPoints(mountPath)
		if len(mountPoints) != 0 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	for _, point := range mountPoints {
		Expect(ensureCleanUp(point)).To(Succeed())
	}
}

func internalMountPoints(mountPath string) []string {
	output, err := exec.Command("mount").Output()
	Expect(err).NotTo(HaveOccurred())

	mountPoints := []string{}
	buffer := bytes.NewBuffer(output)
	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		mountLine := scanner.Text()
		mountInfo := strings.Split(mountLine, " ")
		mountType := mountInfo[0]
		if mountType == "overlay" && strings.Contains(mountLine, mountPath) {
			mountPoint := mountInfo[2]
			mountPoints = append(mountPoints, mountPoint)
		}
	}

	return mountPoints
}

func log(buff *gbytes.Buffer, message string, args ...interface{}) {
	_, err := buff.Write([]byte(fmt.Sprintf(message, args...)))
	Expect(err).NotTo(HaveOccurred())
}

func ensureCleanUp(mountPoint string) error {
	buff := gbytes.NewBuffer()
	log(buff, "Unmounting overlay mountpoint %s\n", mountPoint)
	Expect(syscall.Unmount(mountPoint, 0)).To(Succeed())

	var rmErr error

	defer func() {
		if rmErr != nil {
			fmt.Println("Ensure cleanup loop failed. Details below:", string(buff.Contents()))
		}
	}()

	for i := 0; i < 5; i++ {
		log(buff, "This is #%d rm attempt of mountPoint %s\n", i+1, mountPoint)
		if rmErr = os.RemoveAll(mountPoint); rmErr == nil {
			return nil
		}

		log(buff, "Unmounting overlay mountpoint %s again\n", mountPoint)
		Expect(syscall.Unmount(mountPoint, 0)).To(Succeed())

		time.Sleep(100 * time.Millisecond)
	}

	return rmErr
}

func CleanUpImages(storePath string) {
	files, err := ioutil.ReadDir(filepath.Join(storePath, store.ImageDirName))
	if err != nil {
		return
	}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			Expect(os.RemoveAll(filepath.Join(storePath, store.ImageDirName, fileInfo.Name()))).To(Succeed())
		}
	}
}
