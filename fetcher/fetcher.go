package fetcher // import "code.cloudfoundry.org/grootfs/fetcher"

import "code.cloudfoundry.org/lager"
import digestpkg "github.com/opencontainers/go-digest"

type RemoteBlobFunc func(logger lager.Logger) ([]byte, int64, error)

//go:generate counterfeiter . CacheDriver
type CacheDriver interface {
	FetchBlob(logger lager.Logger, id digestpkg.Digest, remoteBlobFunc RemoteBlobFunc) ([]byte, int64, error)
}
