package factomd

import (
	"encoding/hex"

	"github.com/AdamSLevy/jsonrpc2/v14"
)

var c jsonrpc2.Client

const ENDPOINT = "http://localhost:8088/v2"

type DBlockByHeightResult struct {
	DBlock struct {
		Header struct {
			NetworkID int `json:"networkid"`
		} `json:"header"`
	} `json:"dblock"`
}

type DBlockByHeightResult2 struct {
	Result map[string]interface{} `json:"result"`
}

func IsMainnet() (bool, error) {
	var result DBlockByHeightResult
	err := c.Request(nil, ENDPOINT, "dblock-by-height", struct {
		Height int `json:"height"`
	}{Height: 0}, &result)

	if err != nil {
		return false, err
	}

	return result.DBlock.Header.NetworkID == 4203931042, nil
}

func CommitAndRevealEntry(commit []byte, reveal []byte) {
	// Commit
	err := c.Request(nil, ENDPOINT, "commit-entry", struct {
		Message string `json:"message"`
	}{Message: hex.EncodeToString(commit)}, nil)

	if err != nil {
		return
	}

	// Reveal
	c.Request(nil, ENDPOINT, "reveal-entry", struct {
		Entry string `json:"entry"`
	}{Entry: hex.EncodeToString(reveal)}, nil)
}
