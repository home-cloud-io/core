package communicate

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"

	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
)

type (
	Client interface {
		Listen(logger chassis.Logger, _ chassis.Config)
	}
	client struct {
		logger chassis.Logger
	}
)

const (
	heartbeatRate = 5 * time.Second
)

func New(logger chassis.Logger) Client {
	return &client{
		logger: logger,
	}
}

func (c *client) Listen(logger chassis.Logger, _ chassis.Config) {
	ctx := context.Background()
	logger.Info("starting")
	for {
		client := sdConnect.NewDaemonStreamServiceClient(newInsecureClient(), "http://localhost:2225")
		stream := client.Communicate(ctx)

		// spin off workers
		g, _ := errgroup.WithContext(ctx)
		g.Go(func() error {
			return c.listen(stream)
		})
		g.Go(func() error {
			return c.heartbeat(stream)
		})

		// wait on errors
		if err := g.Wait(); err != nil {
			logger.WithError(err).Error("stream failure")
		}

		time.Sleep(1 * time.Second)
	}
}

func (c *client) listen(stream *connect.BidiStreamForClient[v1.DaemonMessage, v1.ServerMessage]) error {
	for {
		message, err := stream.Receive()
		if err != nil {
			return err
		}
		switch message.Message.(type) {
		case *v1.ServerMessage_Reboot:
			c.logger.Info("reboot command")
		case *v1.ServerMessage_Shutdown:
			c.logger.Info("shutdown command")
		case *v1.ServerMessage_Heartbeat:
			c.logger.Debug("heartbeat received")
		}
	}
}

func (c *client) heartbeat(stream *connect.BidiStreamForClient[v1.DaemonMessage, v1.ServerMessage]) error {
	for {
		err := stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_Heartbeat{},
		})
		if err != nil {
			return err
		}
		time.Sleep(heartbeatRate)
	}
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