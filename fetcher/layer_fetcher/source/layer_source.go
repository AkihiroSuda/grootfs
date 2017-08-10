package source // import "code.cloudfoundry.org/grootfs/fetcher/layer_fetcher/source"

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/Sirupsen/logrus"
	_ "github.com/containers/image/docker"
	manifestpkg "github.com/containers/image/manifest"
	_ "github.com/containers/image/oci/layout"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	digestpkg "github.com/opencontainers/go-digest"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	errorspkg "github.com/pkg/errors"
)

const MAX_BLOB_RETRIES = 3

type LayerSource struct {
	trustedRegistries []string
	username          string
	password          string
}

func NewLayerSource(username, password string, trustedRegistries []string) LayerSource {
	return LayerSource{
		username:          username,
		password:          password,
		trustedRegistries: trustedRegistries,
	}
}

func (s *LayerSource) Manifest(logger lager.Logger, baseImageURL *url.URL) (types.Image, error) {
	logger = logger.Session("fetching-image-manifest", lager.Data{"baseImageURL": baseImageURL})
	logger.Info("starting")
	defer logger.Info("ending")

	img, err := s.image(logger, baseImageURL)
	if err != nil {
		logger.Error("fetching-image-reference-failed", err)
		return nil, errorspkg.Wrap(err, "fetching image reference")
	}

	return convertImage(img)
}

func (s *LayerSource) Blob(logger lager.Logger, baseImageURL *url.URL, digest string) (string, int64, error) {
	logrus.SetOutput(os.Stderr)
	logger = logger.Session("streaming-blob", lager.Data{
		"baseImageURL": baseImageURL,
		"digest":       digest,
	})
	logger.Info("starting")
	defer logger.Info("ending")

	imgSrc, err := s.imageSource(logger, baseImageURL)
	if err != nil {
		return "", 0, err
	}

	blobInfo := types.BlobInfo{Digest: digestpkg.Digest(digest)}

	blob, size, err := s.getBlobWithRetries(imgSrc, blobInfo, logger)
	if err != nil {
		return "", 0, err
	}
	logger.Debug("got-blob-stream", lager.Data{"digest": digest, "size": size})

	blobTempFile, err := ioutil.TempFile("", fmt.Sprintf("blob-%s", digest))
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = blobTempFile.Close() }()

	blobReader := io.TeeReader(blob, blobTempFile)
	if !s.checkCheckSum(logger, blobReader, digest) {
		return "", 0, errorspkg.Errorf("invalid checksum: layer is corrupted `%s`", digest)
	}
	return blobTempFile.Name(), size, nil
}

func (s *LayerSource) getBlobWithRetries(imgSrc types.ImageSource, blobInfo types.BlobInfo, logger lager.Logger) (io.ReadCloser, int64, error) {
	var err error
	for i := 0; i < MAX_BLOB_RETRIES; i++ {
		logger.Info("attempt-get-blob", lager.Data{"attempt": i + 1})
		blob, size, e := imgSrc.GetBlob(blobInfo)
		if e == nil {
			return blob, size, nil
		}
		err = e
	}

	return nil, 0, err
}

func (s *LayerSource) checkCheckSum(logger lager.Logger, data io.Reader, digest string) bool {
	var (
		actualSize int64
		err        error
	)

	hash := sha256.New()
	if actualSize, err = io.Copy(hash, data); err != nil {
		logger.Error("failed-to-hash-data", err)
		return false
	}
	logger.Debug("got-blob", lager.Data{"actualSize": actualSize})

	digestID := strings.Split(digest, ":")[1]
	blobContentsSha := hex.EncodeToString(hash.Sum(nil))
	logger.Debug("checking-checksum", lager.Data{
		"digestIDChecksum":   digestID,
		"downloadedChecksum": blobContentsSha,
	})

	return digestID == blobContentsSha
}

func (s *LayerSource) skipTLSValidation(baseImageURL *url.URL) bool {
	for _, trustedRegistry := range s.trustedRegistries {
		if baseImageURL.Host == trustedRegistry {
			return true
		}
	}

	return false
}

func (s *LayerSource) reference(logger lager.Logger, baseImageURL *url.URL) (types.ImageReference, error) {
	refString := "/"
	if baseImageURL.Host != "" {
		refString += "/" + baseImageURL.Host
	}
	refString += baseImageURL.Path

	logger.Debug("parsing-reference", lager.Data{"refString": refString})
	transport := transports.Get(baseImageURL.Scheme)
	ref, err := transport.ParseReference(refString)
	if err != nil {
		return nil, errorspkg.Wrap(err, "parsing url failed")
	}

	return ref, nil
}

func (s *LayerSource) image(logger lager.Logger, baseImageURL *url.URL) (types.Image, error) {
	ref, err := s.reference(logger, baseImageURL)
	if err != nil {
		return nil, err
	}

	skipTLSValidation := s.skipTLSValidation(baseImageURL)
	logger.Debug("new-image", lager.Data{"skipTLSValidation": skipTLSValidation})
	img, err := ref.NewImage(&types.SystemContext{
		DockerInsecureSkipTLSVerify: skipTLSValidation,
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: s.username,
			Password: s.password,
		},
	})
	if err != nil {
		return nil, errorspkg.Wrap(err, "creating image")
	}

	return img, nil
}

func convertImage(originalImage types.Image) (convertedImage types.Image, err error) {
	convertedImage = originalImage

	_, mimetype, err := originalImage.Manifest()

	if mimetype == manifestpkg.DockerV2Schema1MediaType || mimetype == manifestpkg.DockerV2Schema1SignedMediaType {
		diffIds := []digestpkg.Digest{}
		for _, layer := range originalImage.LayerInfos() {
			diffIds = append(diffIds, layer.Digest)
		}

		options := types.ManifestUpdateOptions{
			ManifestMIMEType: manifestpkg.DockerV2Schema2MediaType,
			InformationOnly: types.ManifestUpdateInformation{
				LayerDiffIDs: diffIds,
			},
		}

		convertedImage, err = originalImage.UpdatedImage(options)
	}

	return
}

func (s *LayerSource) imageSource(logger lager.Logger, baseImageURL *url.URL) (types.ImageSource, error) {
	ref, err := s.reference(logger, baseImageURL)
	if err != nil {
		return nil, err
	}

	skipTLSValidation := s.skipTLSValidation(baseImageURL)

	imgSrc, err := ref.NewImageSource(&types.SystemContext{
		DockerInsecureSkipTLSVerify: skipTLSValidation,
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: s.username,
			Password: s.password,
		},
	}, preferedMediaTypes())
	if err != nil {
		return nil, errorspkg.Wrap(err, "creating image source")
	}
	logger.Debug("new-image-source", lager.Data{"skipTLSValidation": skipTLSValidation})

	return imgSrc, nil
}

func preferedMediaTypes() []string {
	return []string{
		specsv1.MediaTypeImageManifest,
		manifestpkg.DockerV2Schema2MediaType,
	}
}
