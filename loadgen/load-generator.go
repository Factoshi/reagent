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
	gen.stop = make(chan interface{})
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
		WithField("entrySizeRange", config.EntrySizeRange).
		WithField("nbChains", len(config.ChainIDsStr)).
		Info("General load config parsed")

	switch config.Type {
	case "constant":
		var clc ConstantLoadConfig
		mapstructure.Decode(config.Params, &clc)
		if err := clc.isValid(); err != nil {
			return fmt.Errorf("Invalid ConstantLoadConfig: %s", err)
		}

		go lg.runConstantLoad(clc, composer)
	default:
		return fmt.Errorf("Non supported load type: [%s]", config.Type)
	}

	return nil
}

func (lg *LoadGenerator) Stop() {
	log.Info("Stopping load...")
	lg.stop <- true
}
