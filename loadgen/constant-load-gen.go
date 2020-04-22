package loadgen

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/PaulBernier/chockagent/factomd"
)

type ConstantLoadConfig struct {
	EPS float64 `mapstructure:"eps"`
}

func (clc ConstantLoadConfig) isValid() error {
	if clc.EPS <= 0 {
		return fmt.Errorf("Invalid EPS [%f]", clc.EPS)
	}

	return nil
}

func (lg *LoadGenerator) runConstantLoad(config ConstantLoadConfig, composer *RandomEntryComposer) {
	interval := time.Duration(int64(1e6/config.EPS)) * time.Microsecond
	log.WithField("config", fmt.Sprintf("%+v", config)).
		WithField("interval", interval).
		Info("Constant load started")

	start := time.Now()
	maxConcurrency := int64(100)
	var errorCount uint64
	var submitted uint64
	var concurrentGoRoutines int64
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-lg.stop:
			log.WithField("duration", time.Now().Sub(start)).
				WithField("errors", errorCount).
				WithField("error-rate", fmt.Sprintf("%.2f%%", (100*float64(errorCount)/float64(submitted)))).
				Info("Constant load stopped")
			return
		case <-ticker.C:
			// Abort if the entries cannot be inserted fast enough and
			// go routines are starting piling up
			if atomic.LoadInt64(&concurrentGoRoutines) > maxConcurrency {
				log.WithField("duration", time.Now().Sub(start)).
					WithField("errors", errorCount).
					WithField("max-concurrency", maxConcurrency).
					Error("Aborting constant load due to too high concurrency")
				lg.stop = nil
				return
			}

			go func() {
				defer atomic.AddInt64(&concurrentGoRoutines, -1)
				atomic.AddInt64(&concurrentGoRoutines, 1)
				atomic.AddUint64(&submitted, 1)

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
	}
}
