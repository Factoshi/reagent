package factomd

import (
	"encoding/hex"
	"net/url"
	"os"

	_log "github.com/PaulBernier/chockagent/log"

	"github.com/AdamSLevy/jsonrpc2/v14"
)

var (
	c           jsonrpc2.Client
	log         = _log.GetLog()
	rpcEndpoint = "http://localhost:8088/v2"
)

func init() {
	if os.Getenv("FACTOMD_RPC_ENDPOINT") != "" {
		_, err := url.ParseRequestURI(os.Getenv("FACTOMD_RPC_ENDPOINT"))
		if err != nil {
			log.WithError(err).Fatalf("Failed to parse FACTOMD_RPC_ENDPOINT: [%s]",
				os.Getenv("FACTOMD_RPC_ENDPOINT"))
		}
		rpcEndpoint = os.Getenv("FACTOMD_RPC_ENDPOINT")
	}
}

type DBlockByHeightResult struct {
	DBlock struct {
		Header struct {
			NetworkID int `json:"networkid"`
		} `json:"header"`
	} `json:"dblock"`
}

type CurrentMinuteResult struct {
	DBHeight int `json:"directoryblockheight"`
	Minute   int `json:"minute"`
}

func IsMainnet() (bool, error) {
	var result DBlockByHeightResult
	err := c.Request(nil, rpcEndpoint, "dblock-by-height", struct {
		Height int `json:"height"`
	}{Height: 0}, &result)

	if err != nil {
		return false, err
	}

	return result.DBlock.Header.NetworkID == 4203931042, nil
}

func CurrentBlockAndMinute() (int, int, error) {
	var result CurrentMinuteResult
	err := c.Request(nil, rpcEndpoint, "current-minute", nil, &result)

	if err != nil {
		return 0, 0, err
	}

	return result.DBHeight + 1, result.Minute, nil
}

func CommitAndRevealEntry(commit []byte, reveal []byte) error {
	// Commit
	err := c.Request(nil, rpcEndpoint, "commit-entry", struct {
		Message string `json:"message"`
	}{Message: hex.EncodeToString(commit)}, nil)

	if err != nil {
		return err
	}

	// Reveal
	return c.Request(nil, rpcEndpoint, "reveal-entry", struct {
		Entry string `json:"entry"`
	}{Entry: hex.EncodeToString(reveal)}, nil)
}
