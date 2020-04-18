package agent

import (
	"encoding/json"

	_log "github.com/PaulBernier/chockagent/log"

	"github.com/PaulBernier/chockagent/websocket"
)

var (
	log = _log.GetLog()
)

type Agent struct {
	Name  string
	wscli *websocket.Client
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

	go func() {
		defer close(done)
		a.run(stop)
	}()

	return done
}

func (a *Agent) run(stop <-chan struct{}) {
	stopWsCli := make(chan struct{})
	doneServer := a.wscli.Start(a.Name, stopWsCli)

	for {
		select {
		case received, ok := <-a.wscli.Receive:
			if !ok {
				return
			}
			a.handleMessage(received)
		case _, ok := <-a.wscli.Disconnected:
			if !ok {
				return
			}

			// TODO: If lose connection to the master, stop the load
		case <-stop:
			// TODO: stop load gracefully

			// Stop WS server
			close(stopWsCli)
			<-doneServer
			return
		}
	}
}

type Command struct {
	Command string                 `json:"command"`
	Params  map[string]interface{} `json:"params"`
}

func (a *Agent) handleMessage(received []byte) {
	cmd := Command{}
	err := json.Unmarshal(received, &cmd)
	if err != nil {
		log.WithError(err).Error("Failed to unmarshal message")
		return
	}

	log.Infof("Command received: [%s]", cmd.Command)
	switch cmd.Command {
	case "start-load":
		// TODO
	case "stop-load":
		// TODO
	default:
		log.Warnf("Unexpected command [%s]!\n", cmd.Command)
	}
}
