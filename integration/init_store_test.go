package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"syscall"

	"code.cloudfoundry.org/grootfs/groot"
	"code.cloudfoundry.org/grootfs/integration"
	grootfsRunner "code.cloudfoundry.org/grootfs/integration/runner"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Init Store", func() {
	const (
		rootUID = 0
		rootGID = 0
	)

	var (
		runner grootfsRunner.Runner
		spec   grootfsRunner.InitSpec
	)

	BeforeEach(func() {
		integration.SkipIfNonRoot(GrootfsTestUid)

		runner = Runner.WithStore(StorePath).SkipInitStore()
		spec = grootfsRunner.InitSpec{
			FilesystemDriver: Driver,
		}
	})

	Context("when --store-size-bytes is passed", func() {
		var backingStoreFile string

		BeforeEach(func() {
			spec.StoreSizeBytes = 500 * 1024 * 1024
			backingStoreFile = fmt.Sprintf("%s.backing-store", StorePath)
		})

		AfterEach(func() {
			_ = syscall.Unmount(StorePath, 0)
			Expect(os.RemoveAll(backingStoreFile)).To(Succeed())
		})

		It("creates the backing file with the correct size", func() {
			err := runner.InitStore(spec)
			Expect(err).NotTo(HaveOccurred())

			stat, err := os.Stat(backingStoreFile)
			Expect(err).To(BeNil())
			Expect(stat.Size()).To(Equal(int64(500 * 1024 * 1024)))
		})

		It("initialises a filesystem in the backing file", func() {
			Expect(runner.InitStore(spec)).To(Succeed())

			buffer := gbytes.NewBuffer()
			cmd := exec.Command("file", backingStoreFile)
			cmd.Stdout = buffer
			Expect(cmd.Run()).To(Succeed())

			fsType := map[string]string{"btrfs": "BTRFS", "overlay-xfs": "XFS"}
			Expect(buffer).To(gbytes.Say(fsType[Driver]))
		})

		Context("when the given store path is already initialized", func() {
			BeforeEach(func() {
				Expect(runner.InitStore(spec)).To(Succeed())
			})

			It("logs the event", func() {
				logs := gbytes.NewBuffer()
				Expect(runner.WithStderr(logs).InitStore(spec)).To(Succeed())

				Expect(logs).To(gbytes.Say("store-already-initialized"))
			})
		})

		Context("when the given backing store size is too small", func() {
			BeforeEach(func() {
				spec.StoreSizeBytes = 199 * 1024 * 1024
			})

			It("returns an error", func() {
				err := runner.InitStore(spec)
				Expect(err).To(MatchError(ContainSubstring("store size must be at least 200Mb")))
			})
		})

		Context("when using external log device", func() {
			JustBeforeEach(func() {
				runner = runner.WithExternalLogDeviceSize(64)
				integration.SkipIfNotXFS(Driver)
			})

			It("uses the logdev mount option", func() {
				Expect(runner.InitStore(spec)).To(Succeed())

				cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("cat /proc/mounts | grep %s", StorePath))
				output, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred())

				Expect(string(output)).To(ContainSubstring("logdev="))
			})
		})
	})

	Context("when using external log device without store size bytes", func() {
		BeforeEach(func() {
			runner = runner.WithExternalLogDeviceSize(64)
			integration.SkipIfNotXFS(Driver)
		})

		It("returns an error", func() {
			err := runner.InitStore(spec)
			Expect(err).To(MatchError(ContainSubstring("--external-logdev-size-mb requires the --store-size-bytes flag")))
		})
	})

	It("returns a newly created store path", func() {
		Expect(runner.InitStore(spec)).To(Succeed())

		Expect(runner.StorePath).To(BeADirectory())

		stat, err := os.Stat(runner.StorePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(stat.Mode().Perm()).To(Equal(os.FileMode(0700)))
	})

	It("sets the ownership to the caller user", func() {
		Expect(runner.InitStore(spec)).To(Succeed())

		Expect(runner.StorePath).To(BeADirectory())

		stat, err := os.Stat(runner.StorePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(stat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(rootUID)))
		Expect(stat.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(rootGID)))
	})

	Context("when id mappings are provided", func() {
		BeforeEach(func() {
			spec.UIDMappings = []groot.IDMappingSpec{
				groot.IDMappingSpec{HostID: GrootUID, NamespaceID: 0, Size: 1},
				groot.IDMappingSpec{HostID: 100000, NamespaceID: 1, Size: 65000},
			}
			spec.GIDMappings = []groot.IDMappingSpec{
				groot.IDMappingSpec{HostID: GrootGID, NamespaceID: 0, Size: 1},
				groot.IDMappingSpec{HostID: 100000, NamespaceID: 1, Size: 65000},
			}
		})

		It("sets the ownership to the user mapped as root", func() {
			Expect(runner.InitStore(spec)).To(Succeed())

			Expect(runner.StorePath).To(BeADirectory())

			stat, err := os.Stat(runner.StorePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(stat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(GrootUID)))
			Expect(stat.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(GrootGID)))
		})

		It("creates a store that correctly maps the user/group ids", func() {
			sourceImagePath := integration.CreateBaseImage(0, 0, GrootUID, GrootGID)
			baseImageFile := integration.CreateBaseImageTar(sourceImagePath)
			baseImagePath := baseImageFile.Name()

			defer func() {
				Expect(os.RemoveAll(sourceImagePath)).To(Succeed())
				Expect(os.RemoveAll(baseImagePath)).To(Succeed())
			}()

			Expect(runner.InitStore(spec)).To(Succeed())
			image, err := runner.Create(groot.CreateSpec{
				BaseImage: baseImagePath,
				ID:        "random-id",
				Mount:     mountByDefault(),
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(runner.EnsureMounted(image)).To(Succeed())
			grootFi, err := os.Stat(path.Join(image.Rootfs, "foo"))
			Expect(err).NotTo(HaveOccurred())
			Expect(grootFi.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(GrootUID + 99999)))
			Expect(grootFi.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(GrootGID + 99999)))

			rootFi, err := os.Stat(path.Join(image.Rootfs, "bar"))
			Expect(err).NotTo(HaveOccurred())
			Expect(rootFi.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(GrootUID)))
			Expect(rootFi.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(GrootGID)))
		})
	})

	Context("when --rootless is provided", func() {
		BeforeEach(func() {
			spec.Rootless = "groot:groot"
		})

		It("sets the ownership to the provided user and group", func() {
			Expect(runner.InitStore(spec)).To(Succeed())

			Expect(runner.StorePath).To(BeADirectory())

			stat, err := os.Stat(runner.StorePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(stat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(GrootUID)))
			Expect(stat.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(GrootGID)))
		})

		It("creates a store that correctly maps the user/group ids", func() {
			sourceImagePath := integration.CreateBaseImage(0, 0, GrootUID, GrootGID)
			baseImageFile := integration.CreateBaseImageTar(sourceImagePath)
			baseImagePath := baseImageFile.Name()

			defer func() {
				Expect(os.RemoveAll(sourceImagePath)).To(Succeed())
				Expect(os.RemoveAll(baseImagePath)).To(Succeed())
			}()

			Expect(runner.InitStore(spec)).To(Succeed())
			image, err := runner.Create(groot.CreateSpec{
				BaseImage: baseImagePath,
				ID:        "random-id",
				Mount:     mountByDefault(),
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(runner.EnsureMounted(image)).To(Succeed())
			grootFi, err := os.Stat(path.Join(image.Rootfs, "foo"))
			Expect(err).NotTo(HaveOccurred())
			Expect(grootFi.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(GrootUID + 99999)))
			Expect(grootFi.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(GrootGID + 99999)))

			rootFi, err := os.Stat(path.Join(image.Rootfs, "bar"))
			Expect(err).NotTo(HaveOccurred())
			Expect(rootFi.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(GrootUID)))
			Expect(rootFi.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(GrootGID)))
		})

		Context("and id mappings are provided", func() {
			BeforeEach(func() {
				spec.UIDMappings = []groot.IDMappingSpec{
					groot.IDMappingSpec{HostID: GrootUID, NamespaceID: 0, Size: 1},
					groot.IDMappingSpec{HostID: 100000, NamespaceID: 1, Size: 65000},
				}
				spec.GIDMappings = []groot.IDMappingSpec{
					groot.IDMappingSpec{HostID: GrootGID, NamespaceID: 0, Size: 1},
					groot.IDMappingSpec{HostID: 100000, NamespaceID: 1, Size: 65000},
				}
			})

			It("returns an error", func() {
				err := runner.InitStore(spec)
				Expect(err).To(MatchError(ContainSubstring("cannot specify --rootless and --uid-mapping/--gid-mapping")))
			})
		})

		Context("when the rootless parameter is invalid", func() {
			BeforeEach(func() {
				spec.Rootless = "lol"
			})

			It("returns an error", func() {
				err := runner.InitStore(spec)
				Expect(err).To(MatchError(ContainSubstring("invalid --rootless parameter, format must be <user>:<group>")))
			})
		})

		Context("when the user does not exist", func() {
			BeforeEach(func() {
				spec.Rootless = "someoneelse:groot"
			})

			It("returns an error", func() {
				err := runner.InitStore(spec)
				Expect(err).To(MatchError(ContainSubstring("error reading mappings for user 'someoneelse'")))
			})
		})

		Context("when the group does not exist", func() {
			BeforeEach(func() {
				spec.Rootless = "groot:something"
			})

			It("returns an error", func() {
				err := runner.InitStore(spec)
				Expect(err).To(MatchError(ContainSubstring("error reading mappings for group 'something'")))
			})
		})
	})

	Context("when the given driver does not match the mounted path", func() {
		BeforeEach(func() {
			runner = Runner.WithStore("/mnt/ext4/grootfs")
		})

		It("returns an error", func() {
			err := runner.InitStore(spec)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Store path filesystem (/mnt/ext4) is incompatible with requested driver"))
			Expect(runner.StorePath).ToNot(BeADirectory())
		})
	})

	Context("when the given store path is already initialized", func() {
		BeforeEach(func() {
			Expect(runner.InitStore(spec)).To(Succeed())
		})

		It("does not return an error", func() {
			Expect(runner.InitStore(spec)).To(Succeed())
		})
	})

	Context("when the user is not root", func() {
		It("returns an error", func() {
			err := runner.RunningAsUser(GrootUID, GrootUID).InitStore(spec)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("store %s can only be initialized by Root user", runner.StorePath)))
		})
	})
})
