package loadgen

import (
	"fmt"

	"github.com/PaulBernier/chockagent/common"
	_log "github.com/PaulBernier/chockagent/log"

	"github.com/Factom-Asset-Tokens/factom"
	"github.com/mitchellh/mapstructure"
)

var (
	log = _log.GetLog()
)

type LoadGenerator struct {
	stop chan interface{}
}

type LoadConfig struct {
	Type           string
	ChainIDsStr    []string
	EsAddressStr   string
	EntrySizeRange common.IntRange
	Params         map[string]interface{}
}

func NewLoadGenerator() *LoadGenerator {
	gen := new(LoadGenerator)
	return gen
}

func (lg *LoadGenerator) Run(config LoadConfig) error {
	esAddress, err := factom.NewEsAddress(config.EsAddressStr)
	if err != nil {
		return err
	}

	composer, err := NewRandomEntryComposer(config.ChainIDsStr, esAddress, config.EntrySizeRange)
	if err != nil {
		return err
	}

	log.WithField("load-type", config.Type).
		WithField("entry-size-range", config.EntrySizeRange).
		WithField("nb-chains", len(config.ChainIDsStr)).
		Info("General load config parsed")

	switch config.Type {
	case "constant":
		lg.stop = make(chan interface{})

		var clc ConstantLoadConfig
		mapstructure.Decode(config.Params, &clc)
		if err := clc.isValid(); err != nil {
			return fmt.Errorf("Invalid ConstantLoadConfig: %s", err)
		}

		go lg.runConstantLoad(clc, composer)
	case "burst":
		var blc BurstLoadConfig
		mapstructure.Decode(config.Params, &blc)
		if err := blc.isValid(); err != nil {
			return fmt.Errorf("Invalid BurstLoadConfig: %s", err)
		}

		go lg.runBurstLoad(blc, composer)
	default:
		return fmt.Errorf("Non supported load type: [%s]", config.Type)
	}

	return nil
}

func (lg *LoadGenerator) Stop() {
	if lg.stop != nil {
		log.Info("Stopping load...")
		lg.stop <- true
	} else {
		log.Warn("Load already stopped by itself")
	}
}
