package communicate

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/daemon/execute"
	"github.com/home-cloud-io/core/services/platform/daemon/host"
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
	retryLimit    = 25
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

func (c *client) Listen() {
	config := chassis.GetConfig()
	c.logger.Info("starting")
	retries := 0
	for {
		ctx := context.Background()
		if retries > 25 {
			c.logger.Fatal("exhausted retries connecting to server - exiting")
			os.Exit(1)
		}
		client := sdConnect.NewDaemonStreamServiceClient(newInsecureClient(), config.GetString("daemon.server"))
		c.stream = client.Communicate(ctx)

		// spin off workers
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			return c.listen(ctx)
		})
		g.Go(func() error {
			return c.heartbeat()
		})
		g.Go(func() error {
			return c.systemStats(gctx)
		})

		// wait on errors
		if err := g.Wait(); err != nil {
			c.logger.WithError(err).Error("stream failure")
		}

		time.Sleep(1 * time.Second)
		retries++
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
		case *v1.ServerMessage_Heartbeat:
			c.logger.Debug("heartbeat received")
		case *v1.ServerMessage_Restart:
			go restart(ctx, c.logger)
		case *v1.ServerMessage_Shutdown:
			go shutdown(ctx, c.logger)
		case *v1.ServerMessage_RequestOsUpdateDiff:
			go c.osUpdateDiff(ctx)
		case *v1.ServerMessage_RequestCurrentDaemonVersion:
			go c.currentDaemonVersion()
		case *v1.ServerMessage_ChangeDaemonVersionCommand:
			go changeDaemonVersion(ctx, c.logger, message.GetChangeDaemonVersionCommand())
		case *v1.ServerMessage_InstallOsUpdateCommand:
			go installOsUpdate(ctx, c.logger)
		case *v1.ServerMessage_SetSystemImageCommand:
			go setSystemImage(ctx, c.logger, message.GetSetSystemImageCommand())
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
		c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_OsUpdateDiff{
				OsUpdateDiff: &v1.OSUpdateDiff{
					Description: fmt.Sprintf("failed with error: %s", err.Error()),
				},
			},
		})
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
		}
	}
}

func (c *client) currentDaemonVersion() {
	daemonVersion, err := versioning.GetDaemonVersion(c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to get current daemon version")
		c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_CurrentDaemonVersion{
				CurrentDaemonVersion: &v1.CurrentDaemonVersion{
					Version: fmt.Sprintf("failed with error: %s", err.Error()),
				},
			},
		})
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
		}
	}
}

func changeDaemonVersion(ctx context.Context, logger chassis.Logger, def *v1.ChangeDaemonVersionCommand) {
	err := versioning.ChangeDaemonVersion(ctx, logger, def)
	if err != nil {
		logger.WithError(err).Error("failed to change daemon version")
		// TODO: return error to the server?
	}
}

func installOsUpdate(ctx context.Context, logger chassis.Logger) {
	err := versioning.InstallOSUpdate(ctx, logger)
	if err != nil {
		logger.WithError(err).Error("failed to install os update")
		// TODO: return error to the server?
	}
}

func setSystemImage(ctx context.Context, logger chassis.Logger, def *v1.SetSystemImageCommand) {
	err := versioning.SetSystemImage(ctx, logger, def)
	if err != nil {
		logger.WithError(err).Error("failed to set system image")
		// TODO: return error to the server?
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

func (c *client) systemStats(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		go func() {
			stats, err := host.SystemStats([]string{"/"})
			if err != nil {
				c.logger.WithError(err).Error("failed to collect system stats")
			}
			err = c.stream.Send(&v1.DaemonMessage{
				Message: &v1.DaemonMessage_SystemStats{
					SystemStats: stats,
				},
			})
			if err != nil {
				c.logger.WithError(err).Error("failed to send system stats message")
			}
		}()
		time.Sleep(host.ComputeMeasurementDuration)
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
