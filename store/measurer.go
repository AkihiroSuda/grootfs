package store

import (
	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter . VolumeDriver

type VolumeDriver interface {
	Volumes() []string
	VolumeSize(string) (int64, error)
}

type StoreMeasurer struct {
	storePath    string
	volumeDriver VolumeDriver
}

func NewStoreMeasurer(storePath string, volumeDriver VolumeDriver) *StoreMeasurer {
	return &StoreMeasurer{
		storePath:    storePath,
		volumeDriver: volumeDriver,
	}
}

func (s *StoreMeasurer) MeasureStore(logger lager.Logger) (int64, error) {
	logger = logger.Session("measuring-store", lager.Data{"storePath": s.storePath})
	logger.Debug("starting")
	defer logger.Debug("ending")

	usage, err := s.measurePath(s.storePath)
	if err != nil {
		return 0, err
	}

	logger.Debug("store-usage", lager.Data{"bytes": usage})
	return usage, nil
}

func (s *StoreMeasurer) MeasureCache(logger lager.Logger) (int64, error) {
	logger = logger.Session("measuring-cache", lager.Data{"storePath": s.storePath})
	logger.Debug("starting")
	defer logger.Debug("ending")

	usage, err := s.measureCache(s.storePath)
	if err != nil {
		return 0, err
	}

	logger.Debug("cache-usage", lager.Data{"bytes": usage})
	return usage, nil
}
