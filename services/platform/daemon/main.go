package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"

	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
	"golang.org/x/net/http2"
)

func main() {
	logger := zerolog.New()

	runtime := chassis.New(logger)
	defer runtime.Start()

	go func() {
		time.Sleep(1 * time.Second)
		run(logger)
	}()
}

func run(logger chassis.Logger) {
	logger.Info("starting")

	client := sdConnect.NewDaemonStreamServiceClient(newInsecureClient(), "http://localhost:2225")
	stream := client.Communicate(context.Background())

	// listen for messages
	go func() {
		for {
			_, err := stream.Receive()
			if err != nil {
				logger.WithError(err).Error("failed to receive message")
				return
			}
			logger.Info("heartbeat received")
		}
	}()

	// send heartbeats
	go func() {
		for {
			err := stream.Send(&v1.DaemonMessage{
				Message: &v1.DaemonMessage_Heartbeat{},
			})
			if err != nil {
				logger.WithError(err).Error("failed to send message")
				return
			}
			time.Sleep(3 * time.Second)
		}
	}()
}

func newInsecureClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
				// If you're also using this client for non-h2c traffic, you may want
				// to delegate to tls.Dial if the network isn't TCP or the addr isn't
				// in an allowlist.
				return net.Dial(network, addr)
			},
			// Don't forget timeouts!
		},
	}
}
