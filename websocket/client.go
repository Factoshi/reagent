package websocket

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"time"

	_log "github.com/PaulBernier/chockagent/log"
	"github.com/cenkalti/backoff"
	"github.com/gorilla/websocket"
)

var (
	log = _log.GetLog()
)

const (
	pingInterval = 30 * time.Second
)

func exponentialBackOff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     500 * time.Millisecond,
		RandomizationFactor: 0.5,
		Multiplier:          2,
		MaxInterval:         1 * time.Minute,
		MaxElapsedTime:      time.Duration(1<<63 - 1), // For ever
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	return b
}

type Client struct {
	Endpoint string

	Disconnected chan bool

	Send    chan []byte
	Receive chan []byte
}

var chockablockURL string

func init() {
	// Override endpoint with env variable
	if os.Getenv("CHOCKABLOCK_ENDPOINT") != "" {
		_, err := url.ParseRequestURI(os.Getenv("CHOCKABLOCK_ENDPOINT"))
		if err != nil {
			log.WithError(err).Fatalf("Failed to parse CHOCKABLOCK_ENDPOINT: [%s]",
				os.Getenv("CHOCKABLOCK_ENDPOINT"))
		}
		chockablockURL = os.Getenv("CHOCKABLOCK_ENDPOINT")
	} else if chockablockURL == "" {
		// If not set at build time fallback to local dev endpoint
		chockablockURL = "ws://localhost:4007"
	}
}

func NewClient() (cli *Client) {
	cli = new(Client)
	cli.Endpoint = chockablockURL
	cli.Disconnected = make(chan bool)

	cli.Receive = make(chan []byte)
	cli.Send = make(chan []byte)

	return cli
}

func (cli *Client) Start(agentName string, stop <-chan struct{}) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer func() {
			close(cli.Receive)
			close(cli.Send)
			close(cli.Disconnected)
			close(done)
		}()

		conn := cli.connect(agentName, stop)
		doneReading := cli.readPump(conn)
		stopWrite := make(chan struct{})
		cli.writePump(conn, stopWrite)

		for {
			select {
			case err := <-doneReading:
				close(stopWrite)
				cli.Disconnected <- true
				conn.Close()

				if err != nil {
					conn = cli.connect(agentName, stop)
					doneReading = cli.readPump(conn)
					stopWrite = make(chan struct{})
					cli.writePump(conn, stopWrite)
				} else {
					// Graceful shutdown initiated by the server
					return
				}
			case <-stop:
				// Initiate graceful shutdown
				err := conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
					time.Now().Add(10*time.Second),
				)
				close(stopWrite)
				if err != nil {
					log.WithError(err).Error("Failed to gracefully disconnect")
					return
				}

				// Wait for the closing response from the server
				// to shutdown or timeout
				select {
				case <-doneReading:
				case <-time.After(2 * time.Second):
				}
				return
			}
		}
	}()

	return done
}

func (cli *Client) connect(agentName string, stop <-chan struct{}) (conn *websocket.Conn) {
	log.Infof("Connecting to [%s] as [%s]...", cli.Endpoint, agentName)

	header := http.Header{
		"agent_name": []string{agentName},
	}

	ctx, cancel := context.WithCancel(context.Background())
	retryStrategy := exponentialBackOff()
	retryWithContext := backoff.WithContext(retryStrategy, ctx)

	// This goroutine cancels the retries if the stop channel returns anything
	backoffOver := make(chan struct{})
	go func() {
		select {
		case <-stop:
			cancel()
		case <-backoffOver:
		}
	}()

	err := backoff.RetryNotify(func() error {
		d := websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second}
		c, resp, err := d.Dial(cli.Endpoint, header)

		if err != nil {
			return err

			// Fine grained handling of errors below
			// // Server responded but handshake failed
			// if err == websocket.ErrBadHandshake {
			// 	if resp == nil {
			// 		return errors.New("Empty response")
			// 	}
			// } else {
			// 	// Failed to connect
			// 	return err
			// }
		}

		if resp == nil {
			return errors.New("Empty response")
		}

		conn = c

		return nil
	}, retryWithContext, func(err error, duration time.Duration) {
		log.Warnf("Failed to connect. Retrying in %s", duration)
	})
	close(backoffOver)

	if err != nil {
		log.WithError(err).Fatal("Failed to connect")
	}

	log.Info("Connected to ChockaBlock")

	return conn
}

func (cli *Client) readPump(conn *websocket.Conn) (doneReading chan error) {
	doneReading = make(chan error)

	go func() {
		defer close(doneReading)

		for {
			_, message, err := conn.ReadMessage()

			if err != nil {
				// Graceful disconnection
				if e, ok := err.(*websocket.CloseError); ok && e.Code == websocket.CloseNormalClosure {
					if e.Text != "" {
						log.Infof("Disconnection reason: %s", e.Text)
					}
				} else {
					log.WithError(err).Error("Unexpected error reading from server")
					doneReading <- err
				}
				return
			}
			if len(message) > 0 {
				cli.Receive <- message
			}
		}
	}()

	return doneReading
}

func (cli *Client) writePump(conn *websocket.Conn, stopWrite chan struct{}) {
	go func() {
		keepAliveTicker := time.NewTicker(pingInterval)

		defer func() {
			keepAliveTicker.Stop()
		}()

		for {
			select {
			case <-stopWrite:
				return
			case msg, ok := <-cli.Send:
				if !ok {
					return
				}
				err := conn.WriteMessage(websocket.BinaryMessage, msg)
				if err != nil {
					log.WithError(err).Error("Failed to send.")
				}

			case <-keepAliveTicker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(15*time.Second)); err != nil {
					log.WithError(err).Error("Failed to ping server")
				}
			}
		}
	}()
}
