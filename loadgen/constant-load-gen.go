package loadgen

import (
	"fmt"
	"time"

	"github.com/PaulBernier/chockagent/factomd"
)

type ConstantLoadConfig struct {
	EPS            int `mapstructure:"eps"`
	EntrySize      int `mapstructure:"entrySize"`
	EntrySizeRange struct {
		min int `mapstructure:"min"`
		max int `mapstructure:"max"`
	} `mapstructure:"entrySizeRange"`
}

func (clc ConstantLoadConfig) isValid() error {
	if clc.EPS < 1 {
		return fmt.Errorf("Invalid EPS [%d]", clc.EPS)
	}

	if clc.EntrySize == 0 {
		if clc.EntrySizeRange.min < 1 || clc.EntrySizeRange.max < 1 ||
			clc.EntrySizeRange.min > 10240 || clc.EntrySizeRange.max > 10240 ||
			clc.EntrySizeRange.min > clc.EntrySizeRange.max {
			return fmt.Errorf("Invalid EntrySizeRange [%+v]", clc.EntrySizeRange)
		}
	}

	if clc.EntrySizeRange.min == 0 && clc.EntrySizeRange.max == 0 {
		if clc.EntrySize < 1 {
			return fmt.Errorf("Invalid EntrySize: %d", clc.EntrySize)
		}
	}

	return nil
}

func (lg *LoadGenerator) runConstantLoad(config ConstantLoadConfig, composer *RandomEntryComposer) {
	log.Infof("Running constant load with config: [%+v]", config)
	interval := time.Duration(int64(1e6/config.EPS)) * time.Microsecond
	log.Infof("Sending entries at interval [%s]", interval)
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-lg.stop:
			log.Info("Load stopped")
			return
		case <-ticker.C:
			commit, reveal, err := composer.Compose(config.EntrySize)

			if err != nil {
				log.WithError(err).Error("Failed to compose entry")
			} else {
				log.Info(time.Now())
				go factomd.CommitAndRevealEntry(commit, reveal)
			}
		}
	}
}
