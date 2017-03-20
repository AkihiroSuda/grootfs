package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"code.cloudfoundry.org/grootfs/integration"
	"code.cloudfoundry.org/grootfs/integration/runner"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Init Store", func() {
	const rootUID = 0

	var (
		storePath string
		runner    runner.Runner
	)

	BeforeEach(func() {
		integration.SkipIfNonRoot(GrootfsTestUid)
		storePath = filepath.Join(StorePath, "init-store")
		runner = Runner.WithStore(storePath)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(storePath))
	})

	Context("when the given store path is mounted", func() {
		Context("and the given driver matches the mounted path", func() {
			It("returns a newly created store path", func() {
				Expect(runner.InitStore()).To(Succeed())

				Expect(runner.StorePath).To(BeADirectory())

				stat, err := os.Stat(runner.StorePath)
				Expect(err).NotTo(HaveOccurred())
				Expect(stat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(rootUID)))
				Expect(stat.Mode().Perm()).To(Equal(os.FileMode(0755)))
			})
		})

		Context("and the given driver does not match the mounted path", func() {
			BeforeEach(func() {
				runner = Runner.WithStore("/mnt/ext4/grootfs")
			})

			It("returns an error", func() {
				err := runner.InitStore()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Store path filesystem (/mnt/ext4) is incompatible with requested driver"))
				Expect(runner.StorePath).ToNot(BeADirectory())
			})
		})
	})

	Context("when the given store path is already initialized", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(runner.StorePath, 0755)).To(Succeed())
		})

		It("returns an error", func() {
			err := runner.InitStore()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("store already initialized at path %s", runner.StorePath)))
		})
	})

	Context("when the user is not root", func() {
		It("returns an error", func() {
			err := runner.RunningAsUser(GrootUID, GrootUID).InitStore()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("store %s can only be initialized by Root user", runner.StorePath)))
		})
	})
})