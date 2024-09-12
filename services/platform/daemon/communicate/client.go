package communicate

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
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
		mdns   host.DNSPublisher
	}
)

const (
	heartbeatRate = 5 * time.Second
	retryLimit    = 25
)

var (
	clientSingleton Client
)

func NewClient(logger chassis.Logger, mdns host.DNSPublisher) Client {
	clientSingleton = &client{
		logger: logger,
		mdns:   mdns,
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

func (c *client) Send(message *v1.DaemonMessage) error {
	return c.stream.Send(message)
}

// WORKERS

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
			go c.restart(ctx)
		case *v1.ServerMessage_Shutdown:
			go c.shutdown(ctx)
		case *v1.ServerMessage_RequestOsUpdateDiff:
			go c.osUpdateDiff(ctx)
		case *v1.ServerMessage_RequestCurrentDaemonVersion:
			go c.currentDaemonVersion()
		case *v1.ServerMessage_ChangeDaemonVersionCommand:
			go c.changeDaemonVersion(ctx, message.GetChangeDaemonVersionCommand())
		case *v1.ServerMessage_InstallOsUpdateCommand:
			go c.installOsUpdate(ctx)
		case *v1.ServerMessage_SetSystemImageCommand:
			go c.setSystemImage(ctx, message.GetSetSystemImageCommand())
		case *v1.ServerMessage_SetUserPasswordCommand:
			go c.setUserPassword(ctx, message.GetSetUserPasswordCommand())
		case *v1.ServerMessage_SetTimeZoneCommand:
			go c.setTimeZone(ctx, message.GetSetTimeZoneCommand())
		case *v1.ServerMessage_AddMdnsHostCommand:
			go c.addMdnsHost(ctx, message.GetAddMdnsHostCommand())
		case *v1.ServerMessage_RemoveMdnsHostCommand:
			go c.removeMdnsHost(ctx, message.GetRemoveMdnsHostCommand())
		case *v1.ServerMessage_InitializeDeviceCommand:
			go c.initializeDevice(ctx, message.GetInitializeDeviceCommand())
		default:
			c.logger.WithField("message", message).Warn("unknown message type received")
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

// COMMAND HANDLERS

func (c *client) restart(ctx context.Context) {
	c.logger.Info("restart command")
	err := execute.ExecuteCommand(ctx, exec.Command("reboot", "now"))
	if err != nil {
		c.logger.WithError(err).Error("failed to execute restart command")
		// TODO: send error back to server
	}
}

func (c *client) shutdown(ctx context.Context) {
	c.logger.Info("shutdown command")
	err := execute.ExecuteCommand(ctx, exec.Command("shutdown", "now"))
	if err != nil {
		c.logger.WithError(err).Error("failed to execute shutdown command")
		// TODO: send error back to server
	}
}

func (c *client) osUpdateDiff(ctx context.Context) {
	c.logger.Info("os update diff command")
	osUpdateDiff, err := versioning.GetOSVersionDiff(ctx, c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to get os version diff")
		c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_OsUpdateDiff{
				OsUpdateDiff: &v1.OSUpdateDiff{
					Error: &v1.DaemonError{
						Error: err.Error(),
					},
				},
			},
		})
		return
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
	c.logger.Info("finished generating os version diff successfully")
}

func (c *client) currentDaemonVersion() {
	c.logger.Info("current daemon version command")
	current, err := versioning.GetDaemonVersion(c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to get current daemon version")
		c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_CurrentDaemonVersion{
				CurrentDaemonVersion: &v1.CurrentDaemonVersion{
					Error: &v1.DaemonError{
						Error: err.Error(),
					},
				},
			},
		})
		return
	} else {
		err := c.stream.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_CurrentDaemonVersion{
				CurrentDaemonVersion: current,
			},
		})
		if err != nil {
			c.logger.WithError(err).Error("failed to send current daemon version to server")
		}
	}
	c.logger.Info("finished getting current daemon version successfully")
}

func (c *client) changeDaemonVersion(ctx context.Context, def *v1.ChangeDaemonVersionCommand) {
	logger := c.logger.WithFields(chassis.Fields{
		"version":     def.Version,
		"src_hash":    def.SrcHash,
		"vendor_hash": def.VendorHash,
	})
	logger.Info("change daemon version command")
	err := versioning.ChangeDaemonVersion(ctx, c.logger, def)
	if err != nil {
		logger.WithError(err).Error("failed to change daemon version")
		// TODO: return error to the server?
	}
	logger.Info("daemon version changed successfully")
}

func (c *client) installOsUpdate(ctx context.Context) {
	c.logger.Info("install os update command")
	err := versioning.RebuildAndSwitchOS(ctx, c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to install os update")
		// TODO: return error to the server?
	}
}

func (c *client) setSystemImage(ctx context.Context, def *v1.SetSystemImageCommand) {
	logger := c.logger.WithFields(chassis.Fields{
		"current_image":   def.CurrentImage,
		"requested_image": def.RequestedImage,
	})
	logger.Info("set system image command")
	err := versioning.SetSystemImage(ctx, c.logger, def)
	if err != nil {
		logger.WithError(err).Error("failed to set system image")
		// TODO: return error to the server?
		return
	}
	logger.Info("system image set successfully")
}

func (c *client) setUserPassword(ctx context.Context, def *v1.SetUserPasswordCommand) error {
	logger := c.logger.WithField("username", def.Username)

	cmd := exec.Command("chpasswd")

	// write the username:password to stdin when the command executes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		logger.WithError(err).Error("failed to get stdin pipe")
		return err
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, fmt.Sprintf("%s:%s", def.Username, def.Password))
	}()

	// execute command
	err = execute.ExecuteCommand(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to set user password")
		return err
	}
	logger.Info("user password set successfully")
	return nil
}

func (c *client) setTimeZone(ctx context.Context, def *v1.SetTimeZoneCommand) error {
	logger := c.logger.WithField("time_zone", def.TimeZone)
	err := versioning.SetTimeZone(ctx, logger, def.TimeZone)
	if err != nil {
		logger.WithError(err).Error("failed to set time zone")
		return err
	}
	logger.Info("successfully set time zone")
	return nil
}

func (c *client) addMdnsHost(ctx context.Context, def *v1.AddMdnsHostCommand) {
	c.mdns.AddHost(ctx, def.Hostname)
}

func (c *client) removeMdnsHost(_ context.Context, def *v1.RemoveMdnsHostCommand) {
	err := c.mdns.RemoveHost(def.Hostname)
	if err != nil {
		c.logger.WithError(err).Error("failed to remove mDNS host")
	}
}

func (c *client) initializeDevice(ctx context.Context, def *v1.InitializeDeviceCommand) {
	c.logger.Info("initializing device")
	err := c.setUserPassword(ctx, def.User)
	if err != nil {
		err = c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_DeviceInitialized{
				DeviceInitialized: &v1.DeviceInitialized{
					Error: &v1.DaemonError{
						Error: fmt.Sprintf("failed to set user password: %s", err.Error()),
					},
				},
			},
		})
		if err != nil {
			c.logger.WithError(err).Error("failed to send error")
		}
		return
	}
	err = c.setTimeZone(ctx, def.TimeZone)
	if err != nil {
		err = c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_DeviceInitialized{
				DeviceInitialized: &v1.DeviceInitialized{
					Error: &v1.DaemonError{
						Error: fmt.Sprintf("failed to set time zone: %s", err.Error()),
					},
				},
			},
		})
		if err != nil {
			c.logger.WithError(err).Error("failed to send error")
		}
		return
	}

	err = c.Send(&v1.DaemonMessage{
		Message: &v1.DaemonMessage_DeviceInitialized{
			DeviceInitialized: &v1.DeviceInitialized{},
		},
	})
	if err != nil {
		c.logger.WithError(err).Error("failed to send success")
		return
	}

	c.logger.Info("finished initializing device")
}
