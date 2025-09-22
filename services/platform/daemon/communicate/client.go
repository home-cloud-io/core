package communicate

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/daemon/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/daemon/execute"
	"github.com/home-cloud-io/core/services/platform/daemon/host"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
)

type (
	Client interface {
		Listen()
		Send(message *v1.DaemonMessage, request *v1.ServerMessage)
		SendWithError(message *v1.DaemonMessage, request *v1.ServerMessage) error
	}
	client struct {
		mutex           sync.Mutex
		logger          chassis.Logger
		stream          *connect.BidiStreamForClient[v1.DaemonMessage, v1.ServerMessage]
		mdns            host.DNSPublisher
		secureTunneling host.SecureTunnelingController
	}
)

const (
	heartbeatRate = 5 * time.Second
	retryLimit    = 25
)

var (
	clientSingleton Client

	ErrNoStream = fmt.Errorf("no stream")
)

func NewClient(logger chassis.Logger, mdns host.DNSPublisher, secureTunneling host.SecureTunnelingController) Client {
	clientSingleton = &client{
		mutex:           sync.Mutex{},
		logger:          logger,
		mdns:            mdns,
		secureTunneling: secureTunneling,
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
		client := sdConnect.NewDaemonStreamServiceClient(
			newInsecureClient(),
			config.GetString("daemon.server"),
		)
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
		// send the SettingsSaved event to cover the case where the daemon could be restarted while running the `nixos-rebuild switch` command
		g.Go(func() error {
			c.Send(&v1.DaemonMessage{
				Message: &v1.DaemonMessage_SettingsSaved{
					SettingsSaved: &v1.SettingsSaved{},
				},
			}, nil)
			return nil
		})

		// wait on errors
		if err := g.Wait(); err != nil {
			c.logger.WithError(err).Error("stream failure")
		}

		time.Sleep(5 * time.Second)
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
			IdleConnTimeout:  5 * time.Second,
			ReadIdleTimeout:  5 * time.Second,
			WriteByteTimeout: 5 * time.Second,
		},
	}
}

func (c *client) Send(message *v1.DaemonMessage, request *v1.ServerMessage) {
	// default message subject to the same subject as the request
	if request != nil && message.Subject == "" {
		message.Subject = request.Subject
	}

	if c.stream == nil {
		c.logger.WithError(ErrNoStream).Error("failed to send message to server")
		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	err := c.stream.Send(message)
	if err != nil {
		c.logger.WithError(ErrNoStream).Error("failed to send message to server")
		return
	}
}

func (c *client) SendWithError(message *v1.DaemonMessage, request *v1.ServerMessage) error {
	// default message subject to the same subject as the request
	if request != nil && message.Subject == "" {
		message.Subject = request.Subject
	}

	if c.stream == nil {
		return ErrNoStream
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	err := c.stream.Send(message)
	return err
}

// WORKERS

func (c *client) listen(ctx context.Context) error {
	c.logger.Info("listening for messages from server")
	for {
		message, err := c.stream.Receive()
		if err != nil {
			return err
		}
		switch message.Message.(type) {
		case *v1.ServerMessage_Heartbeat:
			c.logger.Trace("heartbeat received")
		case *v1.ServerMessage_Restart:
			go execute.Restart(ctx, c.logger)
		case *v1.ServerMessage_Shutdown:
			go execute.Shutdown(ctx, c.logger)
		case *v1.ServerMessage_RequestOsUpdateDiff:
			go c.osUpdateDiff(ctx, message)
		case *v1.ServerMessage_RequestCurrentDaemonVersion:
			go c.currentDaemonVersion(ctx, message)
		case *v1.ServerMessage_ChangeDaemonVersionCommand:
			go c.changeDaemonVersion(ctx, message.GetChangeDaemonVersionCommand())
		case *v1.ServerMessage_InstallOsUpdateCommand:
			go c.installOsUpdate(ctx)
		case *v1.ServerMessage_SetSystemImageCommand:
			go c.setSystemImage(ctx, message.GetSetSystemImageCommand())
		case *v1.ServerMessage_AddMdnsHostCommand:
			go c.addMdnsHost(ctx, message.GetAddMdnsHostCommand())
		case *v1.ServerMessage_RemoveMdnsHostCommand:
			go c.removeMdnsHost(ctx, message.GetRemoveMdnsHostCommand())
		case *v1.ServerMessage_UploadFileRequest:
			go c.uploadFile(ctx, message)
		case *v1.ServerMessage_SaveSettingsCommand:
			go c.saveSettings(ctx, message)
		case *v1.ServerMessage_AddWireguardInterface:
			go c.addWireguardInterface(ctx, message)
		case *v1.ServerMessage_RemoveWireguardInterface:
			go c.removeWireguardInterface(ctx, message)
		case *v1.ServerMessage_SetStunServerCommand:
			go c.setSTUNServer(ctx, message)
		case *v1.ServerMessage_AddLocatorServerCommand:
			go c.addLocatorServer(ctx, message)
		case *v1.ServerMessage_RemoveLocatorServerCommand:
			go c.removeLocatorServer(ctx, message)
		case *v1.ServerMessage_AddWireguardPeer:
			go c.addWireguardPeer(ctx, message)
		case *v1.ServerMessage_RequestComponentVersionsCommand:
			go c.componentVersions(ctx, message)
		case *v1.ServerMessage_RequestLogsCommand:
			go c.logs(ctx, message)
		default:
			c.logger.WithField("message", message).Warn("unknown message type received")
		}
	}
}

func (c *client) heartbeat() error {
	for {
		err := c.SendWithError(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_Heartbeat{},
		}, nil)
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
			stats, err := host.SystemStats([]string{
				"/",
				host.DataPath(),
			})
			if err != nil {
				c.logger.WithError(err).Error("failed to collect system stats")
			}
			c.Send(&v1.DaemonMessage{
				Message: &v1.DaemonMessage_SystemStats{
					SystemStats: stats,
				},
			}, nil)
		}()
		time.Sleep(host.ComputeMeasurementDuration)
	}
}

// COMMAND HANDLERS

func (c *client) osUpdateDiff(ctx context.Context, msg *v1.ServerMessage) {
	c.logger.Info("os update diff command")
	osUpdateDiff, err := host.GetOSVersionDiff(ctx, c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to get os version diff")
		c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_OsUpdateDiff{
				OsUpdateDiff: &v1.OSUpdateDiff{
					Error: err.Error(),
				},
			},
		}, msg)
		return
	} else {
		c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_OsUpdateDiff{
				OsUpdateDiff: &v1.OSUpdateDiff{
					Description: osUpdateDiff,
				},
			},
		}, msg)
	}
	c.logger.Info("finished generating os version diff successfully")
}

func (c *client) currentDaemonVersion(ctx context.Context, msg *v1.ServerMessage) {
	c.logger.Info("current daemon version command")
	current, err := host.GetDaemonVersion(c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to get current daemon version")
		c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_CurrentDaemonVersion{
				CurrentDaemonVersion: &v1.CurrentDaemonVersion{
					Error: err.Error(),
				},
			},
		}, msg)
		return
	} else {
		c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_CurrentDaemonVersion{
				CurrentDaemonVersion: current,
			},
		}, msg)
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
	err := host.ChangeDaemonVersion(ctx, c.logger, def)
	if err != nil {
		logger.WithError(err).Error("failed to change daemon version")
		// TODO: return error to the server?
	}
	logger.Info("daemon version changed successfully")
}

func (c *client) installOsUpdate(ctx context.Context) {
	c.logger.Info("install os update command")
	err := host.RebuildAndSwitchOS(ctx, c.logger)
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
	err := host.SetSystemImage(ctx, c.logger, def)
	if err != nil {
		logger.WithError(err).Error("failed to set system image")
		// TODO: return error to the server?
		return
	}
	logger.Info("system image set successfully")
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

func (c *client) saveSettings(ctx context.Context, msg *v1.ServerMessage) {
	def := msg.GetSaveSettingsCommand()
	err := host.SaveSettings(ctx, c.logger, def)
	if err != nil {
		c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_SettingsSaved{
				SettingsSaved: &v1.SettingsSaved{
					Error: fmt.Sprintf("failed to save settings: %s", err.Error()),
				},
			},
		}, msg)
		return
	}

	c.Send(&v1.DaemonMessage{
		Message: &v1.DaemonMessage_SettingsSaved{
			SettingsSaved: &v1.SettingsSaved{},
		},
	}, msg)
}

func (c *client) setSTUNServer(ctx context.Context, msg *v1.ServerMessage) {
	def := msg.GetSetStunServerCommand()
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_StunServerSet{
			StunServerSet: &v1.STUNServerSet{
				ServerAddress:      def.ServerAddress,
				WireguardInterface: def.WireguardInterface,
			},
		},
	}

	err := c.secureTunneling.BindSTUNServer(ctx, def.WireguardInterface, def.ServerAddress)
	if err != nil {
		c.logger.WithError(err).Error("failed to bind to new stun server")
		msg := resp.GetStunServerSet()
		msg.Error = err.Error()
	}

	c.Send(resp, msg)
}

func (c *client) addLocatorServer(ctx context.Context, msg *v1.ServerMessage) {
	cmd := msg.GetAddLocatorServerCommand()
	c.logger.WithField("locator_address", cmd.LocatorAddress).WithField("wireguard_interface", cmd.WireguardInterface).Info("adding locator server")
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_LocatorServerAdded{
			LocatorServerAdded: &v1.LocatorServerAdded{
				LocatorAddress:     cmd.LocatorAddress,
				WireguardInterface: cmd.WireguardInterface,
			},
		},
	}

	err := c.secureTunneling.AddLocator(ctx, cmd.WireguardInterface, cmd.LocatorAddress)
	if err != nil {
		c.logger.WithError(err).Error("failed to add locator server")
		msg := resp.GetLocatorServerAdded()
		msg.Error = err.Error()
	}

	c.logger.WithField("locator_address", cmd.LocatorAddress).WithField("wireguard_interface", cmd.WireguardInterface).Info("finished adding locator server")
	c.Send(resp, msg)
}

func (c *client) removeLocatorServer(ctx context.Context, msg *v1.ServerMessage) {
	cmd := msg.GetRemoveLocatorServerCommand()
	c.logger.WithField("locator_address", cmd.LocatorAddress).WithField("wireguard_interface", cmd.WireguardInterface).Info("removing locator server")
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_LocatorServerRemoved{
			LocatorServerRemoved: &v1.LocatorServerRemoved{
				LocatorAddress:     cmd.LocatorAddress,
				WireguardInterface: cmd.WireguardInterface,
			},
		},
	}

	err := c.secureTunneling.RemoveLocator(ctx, cmd.WireguardInterface, cmd.LocatorAddress)
	if err != nil {
		c.logger.WithError(err).Error("failed to remove locator server")
		msg := resp.GetLocatorServerRemoved()
		msg.Error = err.Error()
	}

	c.logger.WithField("locator_address", cmd.LocatorAddress).WithField("wireguard_interface", cmd.WireguardInterface).Info("finished removing locator server")
	c.Send(resp, msg)
}

func (c *client) componentVersions(ctx context.Context, msg *v1.ServerMessage) {

	var (
		components = []*v1.ComponentVersion{}
	)

	daemonVersion, err := host.GetDaemonVersion(c.logger)
	if err != nil {
		components = append(components, &v1.ComponentVersion{
			Name:    "daemon",
			Domain:  "system",
			Version: err.Error(),
		})
	} else {
		components = append(components, &v1.ComponentVersion{
			Name:    "daemon",
			Domain:  "system",
			Version: daemonVersion.Version,
		})
	}

	nixosVersion, err := host.GetNixOSVersion(ctx, c.logger)
	if err != nil {
		components = append(components, &v1.ComponentVersion{
			Name:    "nixos",
			Domain:  "system",
			Version: err.Error(),
		})
	} else {
		components = append(components, &v1.ComponentVersion{
			Name:    "nixos",
			Domain:  "system",
			Version: nixosVersion,
		})
	}

	c.Send(&v1.DaemonMessage{
		Message: &v1.DaemonMessage_ComponentVersions{
			ComponentVersions: &v1.ComponentVersions{
				Components: components,
			},
		},
	}, msg)
}

func (c *client) logs(ctx context.Context, msg *v1.ServerMessage) {
	cmd := msg.GetRequestLogsCommand()

	logs, err := host.DaemonLogs(ctx, c.logger, cmd.SinceSeconds)
	if err != nil {
		c.logger.WithError(err).Error("failed to get daemon logs")
		c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_Logs{
				Logs: &v1.Logs{
					Error: err.Error(),
				},
			},
		}, msg)
		return
	}
	c.Send(&v1.DaemonMessage{
		Message: &v1.DaemonMessage_Logs{
			Logs: &v1.Logs{
				RequestId: cmd.RequestId,
				Logs:      logs,
			},
		},
	}, msg)
}
