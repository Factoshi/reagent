package loadgen

import (
	"fmt"
	"time"

	"github.com/PaulBernier/chockagent/factomd"
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

	i := 0
	start := time.Now()
	for {
		select {
		case <-lg.stop:
			log.Info("Burst load aborted")
			return
		default:
			if i >= config.NbEntries {
				log.WithField("duration", time.Now().Sub(start)).Info("Burst load finished")
				return
			}
			go func() {
				commit, reveal, err := composer.Compose()

				if err != nil {
					log.WithError(err).Error("Failed to compose entry")
				} else {
					factomd.CommitAndRevealEntry(commit, reveal)
				}
			}()
			i++
		}
	}
}
