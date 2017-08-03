package source_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"code.cloudfoundry.org/grootfs/fetcher/layer_fetcher/source"
	"code.cloudfoundry.org/grootfs/testhelpers"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/containers/image/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	digestpkg "github.com/opencontainers/go-digest"
	errorspkg "github.com/pkg/errors"
)

var _ = Describe("Layer source: Docker", func() {
	var (
		trustedRegistries []string
		layerSource       *source.LayerSource

		logger       *lagertest.TestLogger
		baseImageURL *url.URL

		configBlob           string
		expectedLayersDigest []types.BlobInfo
		expectedDiffIds      []digestpkg.Digest
	)

	BeforeEach(func() {
		trustedRegistries = []string{}

		configBlob = "sha256:217f3b4afdf698d639f854d9c6d640903a011413bc7e7bffeabe63c7ca7e4a7d"
		expectedLayersDigest = []types.BlobInfo{
			{
				Digest: "sha256:47e3dd80d678c83c50cb133f4cf20e94d088f890679716c8b763418f55827a58",
				Size:   90,
			},
			{
				Digest: "sha256:7f2760e7451ce455121932b178501d60e651f000c3ab3bc12ae5d1f57614cc76",
				Size:   88,
			},
		}
		expectedDiffIds = []digestpkg.Digest{
			digestpkg.NewDigestFromHex("sha256", "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5"),
			digestpkg.NewDigestFromHex("sha256", "d7c6a5f0d9a15779521094fa5eaf026b719984fb4bfe8e0012bd1da1b62615b0"),
		}

		logger = lagertest.NewTestLogger("test-layer-source")
		var err error
		baseImageURL, err = url.Parse("docker:///cfgarden/empty:v0.1.1")
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		layerSource = source.NewLayerSource("", "", trustedRegistries)
	})

	Describe("Manifest", func() {
		It("fetches the manifest", func() {
			manifest, err := layerSource.Manifest(logger, baseImageURL)
			Expect(err).NotTo(HaveOccurred())

			Expect(manifest.ConfigInfo().Digest.String()).To(Equal(configBlob))

			Expect(manifest.LayerInfos()).To(HaveLen(2))
			Expect(manifest.LayerInfos()[0]).To(Equal(expectedLayersDigest[0]))
			Expect(manifest.LayerInfos()[1]).To(Equal(expectedLayersDigest[1]))
		})

		Context("when the image schema version is 1", func() {
			BeforeEach(func() {
				var err error
				baseImageURL, err = url.Parse("docker:///nginx:1.9")
				Expect(err).NotTo(HaveOccurred())
			})

			FIt("fetches the manifest", func() {
				manifest, err := layerSource.Manifest(logger, baseImageURL)
				Expect(err).NotTo(HaveOccurred())

				Expect(manifest.LayerInfos()).To(HaveLen(8))
				Expect(manifest.LayerInfos()[0]).To(Equal(types.BlobInfo{Digest: "sha256:51f5c6a04d83efd2d45c5fd59537218924bc46705e3de6ffc8bc07b51481610b", Size: -1}))
				Expect(manifest.LayerInfos()[1]).To(Equal(types.BlobInfo{Digest: "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4", Size: -1}))
				Expect(manifest.LayerInfos()[2]).To(Equal(types.BlobInfo{Digest: "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4", Size: -1}))
				Expect(manifest.LayerInfos()[3]).To(Equal(types.BlobInfo{Digest: "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4", Size: -1}))
				Expect(manifest.LayerInfos()[4]).To(Equal(types.BlobInfo{Digest: "sha256:640c8f3d0eb2b84205cc43e312914c4ae464d433089ee1c95042b893eb7af09b", Size: -1}))
				Expect(manifest.LayerInfos()[5]).To(Equal(types.BlobInfo{Digest: "sha256:a4335300aa893de13a747fee474cd982c62539fd8e20e9b5eb21125996140b8a", Size: -1}))
				Expect(manifest.LayerInfos()[6]).To(Equal(types.BlobInfo{Digest: "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4", Size: -1}))
				Expect(manifest.LayerInfos()[7]).To(Equal(types.BlobInfo{Digest: "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4", Size: -1}))

				Expect(manifest.ConfigInfo().Digest.String()).To(Equal("sha256:f0f2e4b0f880c47ef68d8bca346ced37d32712b671412704524ac4162fbf944d"))
			})
		})

		Context("when the image is private", func() {
			BeforeEach(func() {
				var err error
				baseImageURL, err = url.Parse("docker:///cfgarden/private")
				Expect(err).NotTo(HaveOccurred())

				configBlob = "sha256:c2bf00eb303023869c676f91af930a12925c24d677999917e8d52c73fa10b73a"
				expectedLayersDigest[0].Digest = "sha256:dabca1fccc91489bf9914945b95582f16d6090f423174641710083d6651db4a4"
				expectedLayersDigest[1].Digest = "sha256:48ce60c2de08a424e10810c41ec2f00916cfd0f12333e96eb4363eb63723be87"
			})

			Context("when the correct credentials are provided", func() {
				JustBeforeEach(func() {
					layerSource = source.NewLayerSource(RegistryUsername, RegistryPassword, trustedRegistries)
				})

				It("fetches the manifest", func() {
					manifest, err := layerSource.Manifest(logger, baseImageURL)
					Expect(err).NotTo(HaveOccurred())

					Expect(manifest.ConfigInfo().Digest.String()).To(Equal(configBlob))

					Expect(manifest.LayerInfos()).To(HaveLen(2))
					Expect(manifest.LayerInfos()[0]).To(Equal(expectedLayersDigest[0]))
					Expect(manifest.LayerInfos()[1]).To(Equal(expectedLayersDigest[1]))
				})
			})

			Context("when invalid credentials are provided", func() {
				It("returns an error", func() {
					baseImageURL, err := url.Parse("docker:cfgarden/empty:v0.1.0")
					Expect(err).NotTo(HaveOccurred())

					_, err = layerSource.Manifest(logger, baseImageURL)
					Expect(err).To(MatchError(ContainSubstring("parsing url failed")))
				})
			})
		})

		Context("when the image url is invalid", func() {
			It("returns an error", func() {
				baseImageURL, err := url.Parse("docker:cfgarden/empty:v0.1.0")
				Expect(err).NotTo(HaveOccurred())

				_, err = layerSource.Manifest(logger, baseImageURL)
				Expect(err).To(MatchError(ContainSubstring("parsing url failed")))
			})
		})

		Context("when the image does not exist", func() {
			BeforeEach(func() {
				var err error
				baseImageURL, err = url.Parse("docker:///cfgarden/non-existing-image")
				Expect(err).NotTo(HaveOccurred())
			})

			It("wraps the containers/image with a useful error", func() {
				_, err := layerSource.Manifest(logger, baseImageURL)
				Expect(err.Error()).To(MatchRegexp("^fetching image reference"))
			})

			It("logs the original error message", func() {
				_, err := layerSource.Manifest(logger, baseImageURL)
				Expect(err).To(HaveOccurred())

				Expect(logger).To(gbytes.Say("fetching-image-reference-failed"))
				Expect(logger).To(gbytes.Say("unauthorized: authentication required"))
			})
		})
	})

	Describe("Config", func() {
		It("fetches the config", func() {
			manifest, err := layerSource.Manifest(logger, baseImageURL)
			Expect(err).NotTo(HaveOccurred())
			config, err := layerSource.Config(logger, baseImageURL, manifest)
			Expect(err).NotTo(HaveOccurred())

			Expect(config.RootFS.DiffIDs).To(HaveLen(2))
			Expect(config.RootFS.DiffIDs[0]).To(Equal(expectedDiffIds[0]))
			Expect(config.RootFS.DiffIDs[1]).To(Equal(expectedDiffIds[1]))
		})

		Context("when the image is private", func() {
			var manifest types.Image

			BeforeEach(func() {
				var err error
				baseImageURL, err = url.Parse("docker:///cfgarden/private")
				Expect(err).NotTo(HaveOccurred())

				manifest, err = layerSource.Manifest(logger, baseImageURL)
				Expect(err).NotTo(HaveOccurred())

				expectedDiffIds = []digestpkg.Digest{
					digestpkg.NewDigestFromHex("sha256", "780016ca8250bcbed0cbcf7b023c75550583de26629e135a1e31c0bf91fba296"),
					digestpkg.NewDigestFromHex("sha256", "56702ece901015f4f42dc82d1386c5ffc13625c008890d52548ff30dd142838b"),
				}
			})

			Context("when the correct credentials are provided", func() {
				JustBeforeEach(func() {
					layerSource = source.NewLayerSource(RegistryUsername, RegistryPassword, trustedRegistries)
				})

				It("fetches the config", func() {
					config, err := layerSource.Config(logger, baseImageURL, manifest)
					Expect(err).NotTo(HaveOccurred())

					Expect(config.RootFS.DiffIDs).To(HaveLen(2))
					Expect(config.RootFS.DiffIDs[0]).To(Equal(expectedDiffIds[0]))
					Expect(config.RootFS.DiffIDs[1]).To(Equal(expectedDiffIds[1]))
				})
			})

			Context("when invalid credentials are provided", func() {
				It("retuns an error", func() {
					_, err := layerSource.Config(logger, baseImageURL, manifest)
					Expect(err).To(MatchError(ContainSubstring("fetching config blob")))
				})
			})
		})

		Context("when the image url is invalid", func() {
			It("returns an error", func() {
				baseImageURL, err := url.Parse("docker:cfgarden/empty:v0.1.0")
				Expect(err).NotTo(HaveOccurred())

				_, err = layerSource.Config(logger, baseImageURL, nil)
				Expect(err).To(MatchError(ContainSubstring("parsing url failed")))
			})
		})
	})

	Context("when fetching a blob fails temporarily", func() {
		var fakeRegistry *testhelpers.FakeRegistry

		BeforeEach(func() {
			dockerHubUrl, err := url.Parse("https://registry-1.docker.io")
			Expect(err).NotTo(HaveOccurred())
			fakeRegistry = testhelpers.NewFakeRegistry(dockerHubUrl)
			fakeRegistry.Start()

			trustedRegistries = []string{fakeRegistry.Addr()}
			baseImageURL, err = url.Parse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			fakeRegistry.Stop()
		})

		It("retries twice", func() {
			fakeRegistry.FailNextRequests(2)

			_, _, err := layerSource.Blob(logger, baseImageURL, expectedLayersDigest[0].Digest.String())
			Expect(err).NotTo(HaveOccurred())

			Expect(logger.TestSink.LogMessages()).To(
				ContainElement("test-layer-source.streaming-blob.attempt-get-blob"))
		})
	})

	Context("when a private registry is used", func() {
		var fakeRegistry *testhelpers.FakeRegistry

		BeforeEach(func() {
			dockerHubUrl, err := url.Parse("https://registry-1.docker.io")
			Expect(err).NotTo(HaveOccurred())
			fakeRegistry = testhelpers.NewFakeRegistry(dockerHubUrl)
			fakeRegistry.Start()

			baseImageURL, err = url.Parse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			fakeRegistry.Stop()
		})

		It("fails to fetch the manifest", func() {
			_, err := layerSource.Manifest(logger, baseImageURL)
			Expect(err).To(HaveOccurred())
		})

		It("fails to fetch the Config", func() {
			_, err := layerSource.Config(logger, baseImageURL, nil)
			e := errorspkg.Cause(err)
			_, ok := e.(*url.Error)
			Expect(ok).To(BeTrue())
		})

		Context("when the private registry is whitelisted", func() {
			BeforeEach(func() {
				trustedRegistries = []string{fakeRegistry.Addr()}
			})

			It("fetches the manifest", func() {
				manifest, err := layerSource.Manifest(logger, baseImageURL)
				Expect(err).NotTo(HaveOccurred())

				Expect(manifest.LayerInfos()).To(HaveLen(2))
				Expect(manifest.LayerInfos()[0]).To(Equal(expectedLayersDigest[0]))
				Expect(manifest.LayerInfos()[1]).To(Equal(expectedLayersDigest[1]))
			})

			It("fetches the config", func() {
				manifest, err := layerSource.Manifest(logger, baseImageURL)
				Expect(err).NotTo(HaveOccurred())

				config, err := layerSource.Config(logger, baseImageURL, manifest)
				Expect(err).NotTo(HaveOccurred())

				Expect(config.RootFS.DiffIDs).To(HaveLen(2))
				Expect(config.RootFS.DiffIDs[0]).To(Equal(expectedDiffIds[0]))
				Expect(config.RootFS.DiffIDs[1]).To(Equal(expectedDiffIds[1]))
			})

			It("downloads a blob", func() {
				blobPath, size, err := layerSource.Blob(logger, baseImageURL, expectedLayersDigest[0].Digest.String())
				Expect(err).NotTo(HaveOccurred())

				blobReader, err := os.Open(blobPath)
				Expect(err).NotTo(HaveOccurred())

				buffer := gbytes.NewBuffer()
				cmd := exec.Command("tar", "tzv")
				cmd.Stdin = blobReader
				sess, err := gexec.Start(cmd, buffer, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(int64(90)))

				Eventually(buffer).Should(gbytes.Say("hello"))
				Eventually(sess).Should(gexec.Exit(0))
			})

			Context("when using private images", func() {
				var manifest types.Image

				BeforeEach(func() {
					var err error
					baseImageURL, err = url.Parse("docker:///cfgarden/private")
					Expect(err).NotTo(HaveOccurred())

					expectedLayersDigest[0].Digest = "sha256:dabca1fccc91489bf9914945b95582f16d6090f423174641710083d6651db4a4"
					expectedLayersDigest[1].Digest = "sha256:48ce60c2de08a424e10810c41ec2f00916cfd0f12333e96eb4363eb63723be87"

					manifest, err = layerSource.Manifest(logger, baseImageURL)
					Expect(err).NotTo(HaveOccurred())

					expectedDiffIds = []digestpkg.Digest{
						digestpkg.NewDigestFromHex("sha256", "780016ca8250bcbed0cbcf7b023c75550583de26629e135a1e31c0bf91fba296"),
						digestpkg.NewDigestFromHex("sha256", "56702ece901015f4f42dc82d1386c5ffc13625c008890d52548ff30dd142838b"),
					}
				})

				JustBeforeEach(func() {
					layerSource = source.NewLayerSource(RegistryUsername, RegistryPassword, trustedRegistries)
				})

				It("fetches the manifest", func() {
					manifest, err := layerSource.Manifest(logger, baseImageURL)
					Expect(err).NotTo(HaveOccurred())

					Expect(manifest.LayerInfos()).To(HaveLen(2))
					Expect(manifest.LayerInfos()[0]).To(Equal(expectedLayersDigest[0]))
					Expect(manifest.LayerInfos()[1]).To(Equal(expectedLayersDigest[1]))
				})

				It("fetches the config", func() {
					config, err := layerSource.Config(logger, baseImageURL, manifest)
					Expect(err).NotTo(HaveOccurred())

					Expect(config.RootFS.DiffIDs).To(HaveLen(2))
					Expect(config.RootFS.DiffIDs[0]).To(Equal(expectedDiffIds[0]))
					Expect(config.RootFS.DiffIDs[1]).To(Equal(expectedDiffIds[1]))
				})

				It("downloads a blob", func() {
					blobPath, size, err := layerSource.Blob(logger, baseImageURL, expectedLayersDigest[0].Digest.String())
					Expect(err).NotTo(HaveOccurred())

					blobReader, err := os.Open(blobPath)
					Expect(err).NotTo(HaveOccurred())

					buffer := gbytes.NewBuffer()
					cmd := exec.Command("tar", "tzv")
					cmd.Stdin = blobReader
					sess, err := gexec.Start(cmd, buffer, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Expect(size).To(Equal(int64(90)))

					Eventually(buffer).Should(gbytes.Say("hello"))
					Eventually(sess).Should(gexec.Exit(0))
				})
			})
		})
	})

	Describe("Blob", func() {
		It("downloads a blob", func() {
			blobPath, size, err := layerSource.Blob(logger, baseImageURL, expectedLayersDigest[0].Digest.String())
			Expect(err).NotTo(HaveOccurred())

			blobReader, err := os.Open(blobPath)
			Expect(err).NotTo(HaveOccurred())

			buffer := gbytes.NewBuffer()
			cmd := exec.Command("tar", "tzv")
			cmd.Stdin = blobReader
			sess, err := gexec.Start(cmd, buffer, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal(int64(90)))

			Eventually(buffer).Should(gbytes.Say("hello"))
			Eventually(sess).Should(gexec.Exit(0))
		})

		Context("when the image is private", func() {
			BeforeEach(func() {
				var err error
				baseImageURL, err = url.Parse("docker:///cfgarden/private")
				Expect(err).NotTo(HaveOccurred())

				expectedLayersDigest[0].Digest = "sha256:dabca1fccc91489bf9914945b95582f16d6090f423174641710083d6651db4a4"
				expectedLayersDigest[1].Digest = "sha256:48ce60c2de08a424e10810c41ec2f00916cfd0f12333e96eb4363eb63723be87"
			})

			Context("when the correct credentials are provided", func() {
				JustBeforeEach(func() {
					layerSource = source.NewLayerSource(RegistryUsername, RegistryPassword, trustedRegistries)
				})

				It("fetches the config", func() {
					blobPath, size, err := layerSource.Blob(logger, baseImageURL, expectedLayersDigest[0].Digest.String())
					Expect(err).NotTo(HaveOccurred())

					blobReader, err := os.Open(blobPath)
					Expect(err).NotTo(HaveOccurred())

					buffer := gbytes.NewBuffer()
					cmd := exec.Command("tar", "tzv")
					cmd.Stdin = blobReader
					sess, err := gexec.Start(cmd, buffer, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Expect(size).To(Equal(int64(90)))

					Eventually(buffer).Should(gbytes.Say("hello"))
					Eventually(sess).Should(gexec.Exit(0))
				})
			})

			Context("when invalid credentials are provided", func() {
				It("retuns an error", func() {
					_, _, err := layerSource.Blob(logger, baseImageURL, expectedLayersDigest[0].Digest.String())
					Expect(err).To(MatchError(ContainSubstring("Invalid status code returned when fetching blob 401")))
				})
			})
		})

		Context("when the image url is invalid", func() {
			It("returns an error", func() {
				baseImageURL, err := url.Parse("docker:cfgarden/empty:v0.1.0")
				Expect(err).NotTo(HaveOccurred())

				_, _, err = layerSource.Blob(logger, baseImageURL, expectedLayersDigest[0].Digest.String())
				Expect(err).To(MatchError(ContainSubstring("parsing url failed")))
			})
		})

		Context("when the blob does not exist", func() {
			It("returns an error", func() {
				_, _, err := layerSource.Blob(logger, baseImageURL, "sha256:steamed-blob")
				Expect(err).To(MatchError(ContainSubstring("fetching blob 404")))
			})
		})

		Context("when the blob is corrupted", func() {
			var fakeRegistry *testhelpers.FakeRegistry

			BeforeEach(func() {
				dockerHubUrl, err := url.Parse("https://registry-1.docker.io")
				Expect(err).NotTo(HaveOccurred())
				fakeRegistry = testhelpers.NewFakeRegistry(dockerHubUrl)
				layerDigest := strings.Split(expectedLayersDigest[1].Digest.String(), ":")[1]
				fakeRegistry.WhenGettingBlob(layerDigest, 1, func(rw http.ResponseWriter, req *http.Request) {
					_, _ = rw.Write([]byte("bad-blob"))
				})
				fakeRegistry.Start()

				baseImageURL, err = url.Parse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
				Expect(err).NotTo(HaveOccurred())

				trustedRegistries = []string{fakeRegistry.Addr()}
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("returns an error", func() {
				_, _, err := layerSource.Blob(logger, baseImageURL, expectedLayersDigest[1].Digest.String())
				Expect(err).To(MatchError(ContainSubstring("invalid checksum: layer is corrupted")))
			})
		})
	})
})
