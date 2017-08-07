package layer_fetcher_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	"code.cloudfoundry.org/grootfs/base_image_puller"

	fetcherpkg "code.cloudfoundry.org/grootfs/fetcher"
	"code.cloudfoundry.org/grootfs/fetcher/fetcherfakes"
	"code.cloudfoundry.org/grootfs/fetcher/layer_fetcher"
	"code.cloudfoundry.org/grootfs/fetcher/layer_fetcher/layer_fetcherfakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	digestpkg "github.com/opencontainers/go-digest"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("LayerFetcher", func() {
	var (
		fakeCacheDriver   *fetcherfakes.FakeCacheDriver
		fakeSource        *layer_fetcherfakes.FakeSource
		fetcher           *layer_fetcher.LayerFetcher
		logger            *lagertest.TestLogger
		baseImageURL      *url.URL
		gzipedBlobContent []byte
	)

	BeforeEach(func() {
		fakeSource = new(layer_fetcherfakes.FakeSource)
		fakeCacheDriver = new(fetcherfakes.FakeCacheDriver)

		gzipBuffer := bytes.NewBuffer([]byte{})
		gzipWriter := gzip.NewWriter(gzipBuffer)
		_, err := gzipWriter.Write([]byte("hello-world"))
		Expect(err).NotTo(HaveOccurred())
		Expect(gzipWriter.Close()).To(Succeed())
		gzipedBlobContent, err = ioutil.ReadAll(gzipBuffer)
		Expect(err).NotTo(HaveOccurred())

		// by default, the cache driver does not do any caching
		fakeCacheDriver.FetchBlobStub = func(logger lager.Logger, id digestpkg.Digest,
			remoteBlobFunc fetcherpkg.RemoteBlobFunc,
		) ([]byte, int64, error) {
			contents, size, err := remoteBlobFunc(logger)
			if err != nil {
				return nil, 0, err
			}

			return contents, size, nil
		}

		fetcher = layer_fetcher.NewLayerFetcher(fakeSource, fakeCacheDriver)

		logger = lagertest.NewTestLogger("test-layer-fetcher")
		baseImageURL, err = url.Parse("docker:///cfgarden/empty:v0.1.1")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("BaseImageInfo", func() {
		It("fetches the manifest", func() {
			_, err := fetcher.BaseImageInfo(logger, baseImageURL)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeSource.ManifestCallCount()).To(Equal(1))
			_, usedImageURL := fakeSource.ManifestArgsForCall(0)
			Expect(usedImageURL).To(Equal(baseImageURL))
		})

		Context("when fetching the manifest fails", func() {
			BeforeEach(func() {
				fakeSource.ManifestReturns(nil, errors.New("fetching the manifest"))
			})

			It("returns an error", func() {
				_, err := fetcher.BaseImageInfo(logger, baseImageURL)
				Expect(err).To(MatchError(ContainSubstring("fetching the manifest")))
			})
		})

		XIt("returns the correct list of layer digests", func() {
			// manifest := layer_fetcher.Manifest{
			// 	Layers: []layer_fetcher.Layer{
			// 		layer_fetcher.Layer{BlobID: "sha256:47e3dd80d678c83c50cb133f4cf20e94d088f890679716c8b763418f55827a58", Size: 1024},
			// 		layer_fetcher.Layer{BlobID: "sha256:7f2760e7451ce455121932b178501d60e651f000c3ab3bc12ae5d1f57614cc76", Size: 2048},
			// 	},
			// }
			// fakeSource.ManifestReturns(manifest, nil)
			fakeSource.ConfigReturns(specsv1.Image{
				RootFS: specsv1.RootFS{
					DiffIDs: []digestpkg.Digest{
						digestpkg.NewDigestFromHex("sha256", "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5"),
						digestpkg.NewDigestFromHex("sha256", "d7c6a5f0d9a15779521094fa5eaf026b719984fb4bfe8e0012bd1da1b62615b0"),
					},
				},
			}, nil)

			baseImageURL, err := url.Parse("docker:///cfgarden/empty:v0.1.1")
			Expect(err).NotTo(HaveOccurred())

			baseImageInfo, err := fetcher.BaseImageInfo(logger, baseImageURL)
			Expect(err).NotTo(HaveOccurred())

			Expect(baseImageInfo.LayersDigest).To(Equal([]base_image_puller.LayerDigest{
				base_image_puller.LayerDigest{
					BlobID:        "sha256:47e3dd80d678c83c50cb133f4cf20e94d088f890679716c8b763418f55827a58",
					ChainID:       "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5",
					ParentChainID: "",
					Size:          1024,
				},
				base_image_puller.LayerDigest{
					BlobID:        "sha256:7f2760e7451ce455121932b178501d60e651f000c3ab3bc12ae5d1f57614cc76",
					ChainID:       "9242945d3c9c7cf5f127f9352fea38b1d3efe62ee76e25f70a3e6db63a14c233",
					ParentChainID: "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5",
					Size:          2048,
				},
			}))
		})

		It("calls the source", func() {
			_, err := fetcher.BaseImageInfo(logger, baseImageURL)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeSource.ConfigCallCount()).To(Equal(1))
			_, usedImageURL, usedManifest := fakeSource.ConfigArgsForCall(0)
			Expect(usedImageURL).To(Equal(baseImageURL))
			// Expect(usedManifest).To(Equal(manifest))
			fmt.Printf("%+v\n", usedManifest)
			Fail("not implemented")
		})

		Context("when fetching the config fails", func() {
			BeforeEach(func() {
				fakeSource.ConfigReturns(specsv1.Image{}, errors.New("fetching the config"))
			})

			It("returns the error", func() {
				_, err := fetcher.BaseImageInfo(logger, baseImageURL)
				Expect(err).To(MatchError(ContainSubstring("fetching the config")))
			})
		})

		It("returns the correct image config", func() {
			timestamp := time.Time{}.In(time.UTC)
			expectedConfig := specsv1.Image{
				Created: &timestamp,
				RootFS: specsv1.RootFS{
					DiffIDs: []digestpkg.Digest{
						digestpkg.NewDigestFromHex("sha256", "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5"),
						digestpkg.NewDigestFromHex("sha256", "d7c6a5f0d9a15779521094fa5eaf026b719984fb4bfe8e0012bd1da1b62615b0"),
					},
				},
			}
			fakeSource.ConfigReturns(expectedConfig, nil)

			baseImageInfo, err := fetcher.BaseImageInfo(logger, baseImageURL)
			Expect(err).NotTo(HaveOccurred())

			Expect(baseImageInfo.Config).To(Equal(expectedConfig))
		})

		Context("when the config is in the cache", func() {
			var (
				expectedConfig specsv1.Image
				configContents []byte
			)

			BeforeEach(func() {
				timestamp := time.Time{}.In(time.UTC)
				expectedConfig = specsv1.Image{
					Created: &timestamp,
					RootFS: specsv1.RootFS{
						DiffIDs: []digestpkg.Digest{
							digestpkg.NewDigestFromHex("sha256", "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5"),
							digestpkg.NewDigestFromHex("sha256", "d7c6a5f0d9a15779521094fa5eaf026b719984fb4bfe8e0012bd1da1b62615b0"),
						},
					},
				}

				var err error
				configContents, err = json.Marshal(expectedConfig)
				Expect(err).NotTo(HaveOccurred())
				fakeCacheDriver.FetchBlobReturns(configContents, 0, nil)
			})

			JustBeforeEach(func() {
				fakeCacheDriver.FetchBlobReturns(configContents, 0, nil)
			})

			XIt("calls the cache driver", func() {
				// manifest := layer_fetcher.Manifest{
				// 	ConfigCacheKey: "sha256:cached-config",
				// }
				// fakeSource.ManifestReturns(manifest, nil)

				_, err := fetcher.BaseImageInfo(logger, baseImageURL)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCacheDriver.FetchBlobCallCount()).To(Equal(1))
				_, id, _ := fakeCacheDriver.FetchBlobArgsForCall(0)
				Expect(id).To(Equal("sha256:cached-config"))
			})

			It("returns the correct image config", func() {
				baseImageInfo, err := fetcher.BaseImageInfo(logger, baseImageURL)
				Expect(err).NotTo(HaveOccurred())

				Expect(baseImageInfo.Config).To(Equal(expectedConfig))
			})

			Context("when the cache returns a corrupted config", func() {
				BeforeEach(func() {
					configContents = []byte("{invalid: json")
				})

				It("returns an error", func() {
					_, err := fetcher.BaseImageInfo(logger, baseImageURL)
					Expect(err).To(MatchError(ContainSubstring("decoding config from JSON")))
				})
			})
		})

		Context("when the cache fails", func() {
			BeforeEach(func() {
				fakeCacheDriver.FetchBlobReturns(nil, 0, errors.New("failed to return"))
			})

			It("returns the error", func() {
				_, err := fetcher.BaseImageInfo(logger, baseImageURL)
				Expect(err).To(MatchError(ContainSubstring("failed to return")))
			})
		})
	})

	Describe("StreamBlob", func() {
		BeforeEach(func() {
			tmpFile, err := ioutil.TempFile("", "")
			Expect(err).NotTo(HaveOccurred())
			_, err = tmpFile.Write(gzipedBlobContent)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = tmpFile.Close() }()

			fakeSource.BlobReturns(tmpFile.Name(), 0, nil)
		})

		It("uses the source", func() {
			_, _, err := fetcher.StreamBlob(logger, baseImageURL, "sha256:layer-digest")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeSource.BlobCallCount()).To(Equal(1))
			_, usedImageURL, usedDigest := fakeSource.BlobArgsForCall(0)
			Expect(usedImageURL).To(Equal(baseImageURL))
			Expect(usedDigest).To(Equal("sha256:layer-digest"))
		})

		It("returns the stream from the source", func(done Done) {
			stream, _, err := fetcher.StreamBlob(logger, baseImageURL, "sha256:layer-digest")
			Expect(err).NotTo(HaveOccurred())

			contents, err := ioutil.ReadAll(stream)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("hello-world"))

			close(done)
		}, 2.0)

		It("returns the size of the stream", func() {
			tmpFile, err := ioutil.TempFile("", "")
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = tmpFile.Close() }()

			gzipWriter := gzip.NewWriter(tmpFile)
			Expect(gzipWriter.Close()).To(Succeed())

			fakeSource.BlobReturns(tmpFile.Name(), 1024, nil)

			_, size, err := fetcher.StreamBlob(logger, baseImageURL, "sha256:layer-digest")
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal(int64(1024)))
		})

		Context("when the source fails to stream the blob", func() {
			It("returns an error", func() {
				fakeSource.BlobReturns("", 0, errors.New("failed to stream blob"))

				_, _, err := fetcher.StreamBlob(logger, baseImageURL, "sha256:layer-digest")
				Expect(err).To(MatchError(ContainSubstring("failed to stream blob")))
			})
		})
	})
})
