package agent

import (
	"encoding/json"
	"time"

	"github.com/PaulBernier/chockagent/common"
	"github.com/PaulBernier/chockagent/factomd"
	"github.com/PaulBernier/chockagent/loadgen"
	_log "github.com/PaulBernier/chockagent/log"
	"github.com/mitchellh/mapstructure"

	"github.com/PaulBernier/chockagent/websocket"
)

var (
	log = _log.GetLog()
)

type Agent struct {
	Name          string
	wscli         *websocket.Client
	loadGenerator *loadgen.LoadGenerator
}

func NewAgent(name string) *Agent {
	if name == "" {
		log.Fatal("Agent name cannot be empty")
	}

	agent := new(Agent)
	agent.Name = name
	agent.wscli = websocket.NewClient()

	return agent
}

func (a *Agent) Start(stop <-chan struct{}) <-chan struct{} {
	done := make(chan struct{})

	// Verify that the agent was not deployed along a mainnet node
	mainnet, err := factomd.IsMainnet()
	if err != nil {
		log.WithError(err).Fatal("Failed to reach factomd node")
	}
	if mainnet {
		log.Fatal("Chockagent cannot run against a Factom mainnet node")
	}

	go func() {
		defer close(done)
		a.run(stop)
	}()

	return done
}

func (a *Agent) run(stop <-chan struct{}) {
	stopWsCli := make(chan struct{})
	doneServer := a.wscli.Start(a.Name, stopWsCli)
	a.sendCurrentHeight()
	heightUpdateTicker := time.NewTicker(time.Duration(60) * time.Second)

	for {
		select {
		case <-heightUpdateTicker.C:
			a.sendCurrentHeight()
		case received, ok := <-a.wscli.Receive:
			if !ok {
				return
			}
			a.handleMessage(received)
		case _, ok := <-a.wscli.Disconnected:
			// If lose connection to the master, stop any ongoing load to avoid stale state
			a.stopLoad()
			if !ok {
				return
			}
		case <-stop:
			a.stopLoad()

			// Stop WS server
			close(stopWsCli)
			<-doneServer
			return
		}
	}
}

/**********
 * Send
 **********/

type Message struct {
	Type      string      `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

func (a *Agent) sendCurrentHeight() {
	blockheight, _, err := factomd.CurrentBlockAndMinute()
	if err != nil {
		log.Warnf("Failed to send current height because of height fetching: %s", err)
		return
	}

	msg := Message{Type: "blockheight", Timestamp: time.Now().Unix(), Payload: blockheight}
	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Warnf("Failed to send current height because of JSON marshalling: %s", err)
		return
	}
	a.wscli.Send <- bytes
}

/**********
 * Receive
 **********/

type Command struct {
	Command string                 `json:"command"`
	Params  map[string]interface{} `json:"params"`
}

type StartLoadCommand struct {
	Type           string                 `mapstructure:"type"`
	ChainIDs       []string               `mapstructure:"chainIds"`
	EsAddress      string                 `mapstructure:"esAddress"`
	EntrySizeRange common.IntRange        `mapstructure:"entrySizeRange"`
	Params         map[string]interface{} `mapstructure:"params"`
}

func (a *Agent) handleMessage(received []byte) {
	cmd := Command{}
	err := json.Unmarshal(received, &cmd)
	if err != nil {
		log.WithError(err).Error("Failed to unmarshal message")
		return
	}

	log.WithField("cmd", cmd.Command).Infof("Command received")
	switch cmd.Command {
	case "start-load":
		var slc StartLoadCommand
		mapstructure.Decode(cmd.Params, &slc)
		a.startLoad(slc)
	case "stop-load":
		a.stopLoad()
	default:
		log.Warnf("Unexpected command [%s]!\n", cmd.Command)
	}
}

func (a *Agent) startLoad(slc StartLoadCommand) {
	// Stop any stale load that could be still running
	a.stopLoad()

	loadGenerator := loadgen.NewLoadGenerator()
	err := loadGenerator.Run(loadgen.LoadConfig{
		Type:           slc.Type,
		ChainIDsStr:    slc.ChainIDs,
		EsAddressStr:   slc.EsAddress,
		EntrySizeRange: slc.EntrySizeRange,
		Params:         slc.Params,
	})

	if err != nil {
		log.WithError(err).Error("Failed to start load generator")
	} else {
		a.loadGenerator = loadGenerator
	}
}

func (a *Agent) stopLoad() {
	if a.loadGenerator != nil {
		a.loadGenerator.Stop()
		a.loadGenerator = nil
	}
}
