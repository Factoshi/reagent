# Set default endpoint
LDFLAGS_PROD := "-s -w -X github.com/PaulBernier/chockagent/websocket.chockablockURL=wss://chockagent.luciap.ca"

prod:
	go build -trimpath -ldflags $(LDFLAGS_PROD)

local:
	go build -trimpath -ldflags "-s -w"

.PHONY: clean

clean:
	rm -f chockagent
