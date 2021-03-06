package groot

import (
	"time"

	"code.cloudfoundry.org/lager"
	errorspkg "github.com/pkg/errors"
)

//go:generate counterfeiter . Cleaner
type Cleaner interface {
	Clean(logger lager.Logger, cacheSize int64) (bool, error)
}

type cleaner struct {
	storeMeasurer    StoreMeasurer
	garbageCollector GarbageCollector
	locksmith        Locksmith
	metricsEmitter   MetricsEmitter
}

func IamCleaner(locksmith Locksmith, sm StoreMeasurer,
	gc GarbageCollector, metricsEmitter MetricsEmitter,
) *cleaner {
	return &cleaner{
		locksmith:        locksmith,
		storeMeasurer:    sm,
		garbageCollector: gc,
		metricsEmitter:   metricsEmitter,
	}
}

func (c *cleaner) Clean(logger lager.Logger, cacheBytes int64) (noop bool, err error) {
	logger = logger.Session("groot-cleaning")
	logger.Info("starting")

	if cacheBytes < 0 {
		return true, errorspkg.New("cache bytes must be greater than 0")
	}

	defer c.metricsEmitter.TryEmitDurationFrom(logger, MetricImageCleanTime, time.Now())
	defer logger.Info("ending")

	unusedLayerVolumes, unusedLocalVolumes, err := c.garbageCollector.UnusedVolumes(logger)
	if err != nil {
		logger.Error("finding-unused-failed", err)
	}

	if err := c.garbageCollector.MarkUnused(logger, unusedLocalVolumes); err != nil {
		logger.Info("marking-local-volumes-failed-skipping", lager.Data{"localVolumes": unusedLocalVolumes})
	}

	defer func() {
		if err == nil {
			err = c.garbageCollector.Collect(logger)
		}
	}()

	if cacheBytes > 0 {
		if cacheBytes >= c.storeMeasurer.CacheUsage(logger, unusedLayerVolumes) {
			return true, nil
		}
	}

	lockFile, err := c.locksmith.Lock(GlobalLockKey)
	if err != nil {
		return false, errorspkg.Wrap(err, "garbage collector acquiring lock")
	}

	if err := c.garbageCollector.MarkUnused(logger, unusedLayerVolumes); err != nil {
		logger.Error("marking-unused-failed", err)
	}

	if err := c.locksmith.Unlock(lockFile); err != nil {
		logger.Error("unlocking-failed", err)
	}

	return false, err
}
