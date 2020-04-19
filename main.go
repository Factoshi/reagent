package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PaulBernier/chockagent/agent"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	stopAgent := make(chan struct{})
	agent := agent.NewAgent(os.Getenv("AGENT_NAME"))
	agentDone := agent.Start(stopAgent)

	defer func() {
		close(stopAgent) // Stop agent
		fmt.Println("\nWaiting for Agent to stop...")
		<-agentDone // Wait for orax cli to stop.
		fmt.Printf("Agent stopped.\n\n")
	}()

	defer signal.Reset()
	// Wait for interrupt signal or unexpected termination of orax cli
	select {
	case <-sigs:
	case <-agentDone: // Closed if Agent exits by itself
	}
}
