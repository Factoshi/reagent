package loadgen

import (
	"fmt"
	"time"

	"github.com/PaulBernier/chockagent/factomd"
)

type ConstantLoadConfig struct {
	EPS int `mapstructure:"eps"`
}

func (clc ConstantLoadConfig) isValid() error {
	if clc.EPS < 1 {
		return fmt.Errorf("Invalid EPS [%d]", clc.EPS)
	}

	return nil
}

func (lg *LoadGenerator) runConstantLoad(config ConstantLoadConfig, composer *RandomEntryComposer) {
	interval := time.Duration(int64(1e6/config.EPS)) * time.Microsecond
	log.WithField("config", fmt.Sprintf("%+v", config)).
		WithField("interval", interval).
		Info("Constant load started")

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-lg.stop:
			log.Info("Constant load stopped")
			return
		case <-ticker.C:
			// TODO: add a mechanism to detect when the machine is not able to create load fast enough
			// and goroutines start piling up
			go func() {
				commit, reveal, err := composer.Compose()

				if err != nil {
					log.WithError(err).Error("Failed to compose entry")
				} else {
					factomd.CommitAndRevealEntry(commit, reveal)
				}
			}()
		}
	}
}
