package loadgen

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/PaulBernier/chockagent/factomd"
	"golang.org/x/sync/semaphore"
)

type BurstLoadConfig struct {
	NbEntries int `mapstructure:"nbEntries"`
}

func (blc BurstLoadConfig) isValid() error {
	if blc.NbEntries < 1 {
		return fmt.Errorf("Invalid NbEntries [%d]", blc.NbEntries)
	}

	return nil
}

func (lg *LoadGenerator) runBurstLoad(config BurstLoadConfig, composer *RandomEntryComposer) {
	log.WithField("config", fmt.Sprintf("%+v", config)).
		Info("Burst load started")

	// Limiting API calls concurrency
	concurrency := int64(200)
	sem := semaphore.NewWeighted(concurrency)
	timeout := 10 * time.Second
	start := time.Now()
	var errorCount uint64

	for i := 0; i < config.NbEntries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		if err := sem.Acquire(ctx, 1); err != nil {
			cancel()
			log.WithError(err).Error("Burst aborted")
			return
		}
		cancel()

		go func() {
			defer sem.Release(1)

			commit, reveal, err := composer.Compose()

			// This should never happen, it's a hard failure
			if err != nil {
				atomic.AddUint64(&errorCount, 1)
				log.WithError(err).Error("Fatal: failed to compose entry")
				return
			}

			err = factomd.CommitAndRevealEntry(commit, reveal)
			// It is expected that API calls will start failing under heavy load
			if err != nil {
				atomic.AddUint64(&errorCount, 1)
				log.WithError(err).Warn("Failed to submit entry")
			}

		}()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// Wait for the semaphore to be fully released
	// (to account for the full duration as well as all errors)
	if err := sem.Acquire(ctx, concurrency); err != nil {
		cancel()
		log.WithError(err).Error("Failed to wait for the burst to finish")
		return
	}
	cancel()

	duration := time.Now().Sub(start)
	log.WithField("duration", duration).
		WithField("eps", fmt.Sprintf("%.2f", float64(config.NbEntries)/duration.Seconds())).
		WithField("errors", errorCount).
		WithField("error-rate", fmt.Sprintf("%.2f%%", (100*float64(errorCount)/float64(config.NbEntries)))).
		Info("Burst load finished")

}
