package integration_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"code.cloudfoundry.org/grootfs/groot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("Clean", func() {
	BeforeEach(func() {
		createImages("docker:///cfgarden/empty:v0.1.1", "a")
		_, err := Runner.Create(groot.CreateSpec{
			ID:        "busybox",
			BaseImage: "docker:///busybox:latest",
			Mount:     mountByDefault(),
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(Runner.Delete("busybox")).To(Succeed())
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
		b.Time("first-clean", func() {
			_, err := Runner.Clean(0, []string{})
			Expect(err).NotTo(HaveOccurred())
		})

		deleteAllImages()

		b.Time("second-clean", func() {
			_, err := Runner.Clean(0, []string{})
			Expect(err).NotTo(HaveOccurred())
		})
	}, 10)

})

func createImages(image string, prefix string) {
	for i := 0; i < 199; i++ {
		_, err := Runner.Create(groot.CreateSpec{
			ID:        fmt.Sprintf("my-image-%s-%d", prefix, i),
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
