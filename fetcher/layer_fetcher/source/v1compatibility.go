package source // import "code.cloudfoundry.org/grootfs/fetcher/layer_fetcher/source"

import (
	"github.com/containers/image/docker/reference"
	manifestpkg "github.com/containers/image/manifest"
	"github.com/containers/image/types"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

type image struct {
	wrappedImage types.Image
}

func v1CompatibleImage(imageToWrap types.Image) (types.Image, error) {
	imageWrapper := &image{imageToWrap}

	_, mimetype, _ := imageWrapper.wrappedImage.Manifest()

	if mimetype == manifestpkg.DockerV2Schema1MediaType || mimetype == manifestpkg.DockerV2Schema1SignedMediaType {
		// diffIds := []digest.Digest{}
		// for _, layer := range imageWrapper.wrappedImage.LayerInfos() {
		// 	diffIds = append(diffIds, layer.Digest)
		// }

		options := types.ManifestUpdateOptions{
			ManifestMIMEType: manifestpkg.DockerV2Schema2MediaType,
			// InformationOnly: types.ManifestUpdateInformation{
			// 	LayerDiffIDs: diffIds,
			// },
		}

		var err error
		imageWrapper.wrappedImage, err = imageWrapper.wrappedImage.UpdatedImage(options)
		if err != nil {
			return nil, err
		}
	}

	return imageWrapper, nil
}

func (i *image) ConfigInfo() types.BlobInfo {
	return i.wrappedImage.ConfigInfo()
}

func (i *image) ConfigBlob() ([]byte, error) {
	return i.wrappedImage.ConfigBlob()
}

func (i *image) OCIConfig() (*v1.Image, error) {
	return i.wrappedImage.OCIConfig()
}

func (i *image) LayerInfos() []types.BlobInfo {
	return i.wrappedImage.LayerInfos()
}

func (i *image) EmbeddedDockerReferenceConflicts(ref reference.Named) bool {
	return i.wrappedImage.EmbeddedDockerReferenceConflicts(ref)
}

func (i *image) Inspect() (*types.ImageInspectInfo, error) {
	return i.wrappedImage.Inspect()
}

func (i *image) UpdatedImageNeedsLayerDiffIDs(options types.ManifestUpdateOptions) bool {
	return i.wrappedImage.UpdatedImageNeedsLayerDiffIDs(options)
}

func (i *image) UpdatedImage(options types.ManifestUpdateOptions) (types.Image, error) {
	return i.wrappedImage.UpdatedImage(options)
}

func (i *image) IsMultiImage() bool {
	return i.wrappedImage.IsMultiImage()
}

func (i *image) Size() (int64, error) {
	return i.wrappedImage.Size()
}

func (i *image) Reference() types.ImageReference {
	return i.wrappedImage.Reference()
}

func (i *image) Close() error {
	return i.wrappedImage.Close()
}

func (i *image) Manifest() ([]byte, string, error) {
	return i.wrappedImage.Manifest()
}

func (i *image) Signatures() ([][]byte, error) {
	return i.wrappedImage.Signatures()
}
