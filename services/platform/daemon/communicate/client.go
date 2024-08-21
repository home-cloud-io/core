package communicate

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/daemon/execute"
	"github.com/home-cloud-io/core/services/platform/daemon/versioning"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
)

type (
	Client interface {
		Listen()
		Send(*v1.DaemonMessage) error
	}
	client struct {
		logger chassis.Logger
		stream *connect.BidiStreamForClient[v1.DaemonMessage, v1.ServerMessage]
	}
)

const (
	heartbeatRate = 5 * time.Second
)

var (
	clientSingleton Client
)

func NewClient(logger chassis.Logger) Client {
	clientSingleton = &client{
		logger: logger,
	}
	return clientSingleton
}

func GetClient() Client {
	return clientSingleton
}

func (c *client) Listen() {
	ctx := context.Background()
	config := chassis.GetConfig()
	c.logger.Info("starting")
	for {
		client := sdConnect.NewDaemonStreamServiceClient(newInsecureClient(), config.GetString("daemon.server"))
		c.stream = client.Communicate(ctx)

		// spin off workers
		g, _ := errgroup.WithContext(ctx)
		g.Go(func() error {
			return c.listen(ctx)
		})
		g.Go(func() error {
			return c.heartbeat()
		})

		// wait on errors
		if err := g.Wait(); err != nil {
			c.logger.WithError(err).Error("stream failure")
		}

		time.Sleep(1 * time.Second)
	}
}

func (c *client) Send(message *v1.DaemonMessage) error {
	return c.stream.Send(message)
}

func (c *client) listen(ctx context.Context) error {
	for {
		message, err := c.stream.Receive()
		if err != nil {
			return err
		}
		switch message.Message.(type) {
		case *v1.ServerMessage_Restart:
			go restart(ctx, c.logger)
		case *v1.ServerMessage_Shutdown:
			go shutdown(ctx, c.logger)
		case *v1.ServerMessage_Heartbeat:
			c.logger.Debug("heartbeat received")
		case *v1.ServerMessage_RequestOsUpdateDiff:
			go c.osUpdateDiff(ctx)
		case *v1.ServerMessage_RequestCurrentDaemonVersion:
			go c.currentDaemonVersion()
		default:
			c.logger.WithField("message", message).Warn("unknown message type received")
		}
	}
}

func restart(ctx context.Context, logger chassis.Logger) {
	logger.Info("restart command")
	err := execute.ExecuteCommand(ctx, exec.Command("reboot", "now"))
	if err != nil {
		logger.WithError(err).Error("failed to execute restart command")
		// TODO: send error back to server
	}
}

func shutdown(ctx context.Context, logger chassis.Logger) {
	logger.Info("shutdown command")
	err := execute.ExecuteCommand(ctx, exec.Command("shutdown", "now"))
	if err != nil {
		logger.WithError(err).Error("failed to execute shutdown command")
		// TODO: send error back to server
	}
}

func (c *client) osUpdateDiff(ctx context.Context) {
	osUpdateDiff, err := versioning.GetOSVersionDiff(ctx, c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to get os version diff")
	} else {
		err := c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_OsUpdateDiff{
				OsUpdateDiff: &v1.OSUpdateDiff{
					Description: osUpdateDiff,
				},
			},
		})
		if err != nil {
			c.logger.WithError(err).Error("failed to send os update diff to server")
			c.stream.Send(&v1.DaemonMessage{
				Message: &v1.DaemonMessage_OsUpdateDiff{
					OsUpdateDiff: &v1.OSUpdateDiff{
						Description: fmt.Sprintf("failed with error: %s", err.Error()),
					},
				},
			})
		}
	}
}

func (c *client) currentDaemonVersion() {
	daemonVersion, err := versioning.GetDaemonVersion(c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to get current daemon version")
	} else {
		err := c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_CurrentDaemonVersion{
				CurrentDaemonVersion: &v1.CurrentDaemonVersion{
					Version: daemonVersion,
				},
			},
		})
		if err != nil {
			c.logger.WithError(err).Error("failed to send current daemon version to server")
			c.stream.Send(&v1.DaemonMessage{
				Message: &v1.DaemonMessage_CurrentDaemonVersion{
					CurrentDaemonVersion: &v1.CurrentDaemonVersion{
						Version: fmt.Sprintf("failed with error: %s", err.Error()),
					},
				},
			})
		}
	}
}

func (c *client) heartbeat() error {
	for {
		err := c.stream.Send(&v1.DaemonMessage{
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
