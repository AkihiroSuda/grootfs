package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/grootfs/groot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("Clean", func() {
	BeforeEach(func() {
		workDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())

		images := []string{
			"docker:///cfgarden/empty:v0.1.1",
			"docker:///onsi/grace-busybox:latest",
			"docker:///busybox:latest",
			fmt.Sprintf("oci:///%s/assets/oci-test-image/grootfs-busybox:latest", workDir),
			fmt.Sprintf("oci:///%s/assets/oci-test-image/garden-busybox:latest", workDir),
			fmt.Sprintf("oci:///%s/assets/oci-test-image/empty", workDir),
			fmt.Sprintf("oci:///%s/assets/oci-test-image/non-writable-file:latest", workDir),
			fmt.Sprintf("oci:///%s/assets/oci-test-image/non-writable-folder:latest", workDir),
			fmt.Sprintf("oci:///%s/assets/oci-test-image/opq-whiteouts-busybox:latest", workDir),
		}
		createImages(images)
		deleteAllImages()
	})

	AfterEach(func() {
		images, err := Runner.List()
		Expect(err).NotTo(HaveOccurred())
		Expect(images).To(HaveLen(1))
		Expect(images[0].Path).To(Equal("Store empty"))

		files, err := ioutil.ReadDir(fmt.Sprintf("/mnt/xfs-%d/store/volumes", GinkgoParallelNode()))
		Expect(err).NotTo(HaveOccurred())
		Expect(files).To(BeEmpty())
	})

	Measure("clean time", func(b Benchmarker) {
		b.Time("clean", func() {
			_, err := Runner.Clean(0, []string{})
			Expect(err).NotTo(HaveOccurred())
		})
	}, 10)

})

func createImages(images []string) {
	for i, image := range images {
		_, err := Runner.Create(groot.CreateSpec{
			ID:        fmt.Sprintf("my-image-%d", i),
			BaseImage: image,
			Mount:     mountByDefault(),
		})
		Expect(err).NotTo(HaveOccurred())
	}

}

func deleteAllImages() {
	images, err := Runner.List()
	Expect(err).NotTo(HaveOccurred())

	for _, image := range images {
		imageName := filepath.Base(image.Path)
		Expect(Runner.Delete(imageName)).To(Succeed())
	}
}
