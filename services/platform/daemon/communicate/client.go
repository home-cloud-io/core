package communicate

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
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
		Send(*v1.DaemonMessage)
	}
	client struct {
		mutex             sync.Mutex
		logger            chassis.Logger
		stream            *connect.BidiStreamForClient[v1.DaemonMessage, v1.ServerMessage]
		mdns              host.DNSPublisher
		stun              host.STUNClient
		locatorController host.LocatorController
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

func NewClient(logger chassis.Logger, mdns host.DNSPublisher, stun host.STUNClient, locatorController host.LocatorController) Client {
	clientSingleton = &client{
		mutex:             sync.Mutex{},
		logger:            logger,
		mdns:              mdns,
		stun:              stun,
		locatorController: locatorController,
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
			})
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
		},
	}
}

func (c *client) Send(message *v1.DaemonMessage) {
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

func (c *client) SendWithError(message *v1.DaemonMessage) error {
	if c.stream == nil {
		return ErrNoStream
	}
	c.mutex.Lock()
	err := c.stream.Send(message)
	c.mutex.Unlock()
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
		case *v1.ServerMessage_AddMdnsHostCommand:
			go c.addMdnsHost(ctx, message.GetAddMdnsHostCommand())
		case *v1.ServerMessage_RemoveMdnsHostCommand:
			go c.removeMdnsHost(ctx, message.GetRemoveMdnsHostCommand())
		case *v1.ServerMessage_UploadFileRequest:
			go c.uploadFile(ctx, message.GetUploadFileRequest())
		case *v1.ServerMessage_SaveSettingsCommand:
			go c.saveSettings(ctx, message.GetSaveSettingsCommand())
		case *v1.ServerMessage_AddWireguardInterface:
			go c.addWireguardInterface(ctx, message.GetAddWireguardInterface())
		case *v1.ServerMessage_RemoveWireguardInterface:
			go c.removeWireguardInterface(ctx, message.GetRemoveWireguardInterface())
		case *v1.ServerMessage_SetStunServerCommand:
			go c.setSTUNServer(ctx, message.GetSetStunServerCommand())
		case *v1.ServerMessage_AddLocatorServerCommand:
			go c.addLocatorServer(ctx, message.GetAddLocatorServerCommand())
		case *v1.ServerMessage_RemoveLocatorServerCommand:
			go c.removeLocatorServer(ctx, message.GetRemoveLocatorServerCommand())
		case *v1.ServerMessage_DisableAllLocatorsCommand:
			go c.disableAllLocators(ctx, message.GetDisableAllLocatorsCommand())
		case *v1.ServerMessage_AddWireguardPeer:
			go c.addWireguardPeer(ctx, message.GetAddWireguardPeer().GetPeer())
		default:
			c.logger.WithField("message", message).Warn("unknown message type received")
		}
	}
}

func (c *client) heartbeat() error {
	for {
		err := c.SendWithError(&v1.DaemonMessage{
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
			stats, err := host.SystemStats([]string{host.DataPath()})
			if err != nil {
				c.logger.WithError(err).Error("failed to collect system stats")
			}
			c.Send(&v1.DaemonMessage{
				Message: &v1.DaemonMessage_SystemStats{
					SystemStats: stats,
				},
			})
		}()
		time.Sleep(host.ComputeMeasurementDuration)
	}
}

// COMMAND HANDLERS

func (c *client) restart(ctx context.Context) {
	c.logger.Info("restart command")
	if chassis.GetConfig().Env() == "test" {
		c.logger.Info("mocking restart")
		return
	}
	err := execute.ExecuteCommand(ctx, exec.Command("reboot", "now"))
	if err != nil {
		c.logger.WithError(err).Error("failed to execute restart command")
		// TODO: send error back to server
	}
}

func (c *client) shutdown(ctx context.Context) {
	c.logger.Info("shutdown command")
	if chassis.GetConfig().Env() == "test" {
		c.logger.Info("mocking shutdown")
		return
	}
	err := execute.ExecuteCommand(ctx, exec.Command("shutdown", "now"))
	if err != nil {
		c.logger.WithError(err).Error("failed to execute shutdown command")
		// TODO: send error back to server
	}
}

func (c *client) osUpdateDiff(ctx context.Context) {
	c.logger.Info("os update diff command")
	osUpdateDiff, err := host.GetOSVersionDiff(ctx, c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to get os version diff")
		c.Send(&v1.DaemonMessage{
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
		c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_OsUpdateDiff{
				OsUpdateDiff: &v1.OSUpdateDiff{
					Description: osUpdateDiff,
				},
			},
		})
	}
	c.logger.Info("finished generating os version diff successfully")
}

func (c *client) currentDaemonVersion() {
	c.logger.Info("current daemon version command")
	current, err := host.GetDaemonVersion(c.logger)
	if err != nil {
		c.logger.WithError(err).Error("failed to get current daemon version")
		c.Send(&v1.DaemonMessage{
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
		c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_CurrentDaemonVersion{
				CurrentDaemonVersion: current,
			},
		})
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

func (c *client) saveSettings(ctx context.Context, def *v1.SaveSettingsCommand) {
	err := host.SaveSettings(ctx, c.logger, def)
	if err != nil {
		c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_SettingsSaved{
				SettingsSaved: &v1.SettingsSaved{
					Error: &v1.DaemonError{
						Error: fmt.Sprintf("failed to save settings: %s", err.Error()),
					},
				},
			},
		})
		return
	}

	c.Send(&v1.DaemonMessage{
		Message: &v1.DaemonMessage_SettingsSaved{
			SettingsSaved: &v1.SettingsSaved{},
		},
	})
}

func (c *client) setSTUNServer(_ context.Context, def *v1.SetSTUNServerCommand) {
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_StunServerSet{
			StunServerSet: &v1.STUNServerSet{},
		},
	}

	_, err := c.stun.Bind(def.Server)
	if err != nil {
		c.logger.WithError(err).Error("failed to bind to new stun server")
		resp.GetStunServerSet().Error = &v1.DaemonError{Error: err.Error()}
	}

	c.Send(resp)
}

func (c *client) addLocatorServer(ctx context.Context, cmd *v1.AddLocatorServerCommand) {
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_LocatorServerAdded{
			LocatorServerAdded: &v1.LocatorServerAdded{},
		},
	}

	locator, err := c.locatorController.AddLocator(ctx, cmd.LocatorAddress)
	if err != nil {
		c.logger.WithError(err).Error("failed to add locator server")
		resp.GetLocatorServerAdded().Error = &v1.DaemonError{Error: err.Error()}
	}
	resp.GetLocatorServerAdded().Locator = locator

	c.Send(resp)
}

func (c *client) removeLocatorServer(ctx context.Context, cmd *v1.RemoveLocatorServerCommand) {
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_LocatorServerRemoved{
			LocatorServerRemoved: &v1.LocatorServerRemoved{
				Address: cmd.LocatorAddress,
			},
		},
	}

	err := c.locatorController.RemoveLocator(ctx, cmd.LocatorAddress)
	if err != nil {
		c.logger.WithError(err).Error("failed to remove locator server")
		resp.GetLocatorServerRemoved().Error = &v1.DaemonError{Error: err.Error()}
	}

	c.Send(resp)
}

func (c *client) disableAllLocators(ctx context.Context, cmd *v1.DisableAllLocatorsCommand) {
	resp := &v1.DaemonMessage{
		Message: &v1.DaemonMessage_AllLocatorsDisabled{
			AllLocatorsDisabled: &v1.AllLocatorsDisabled{},
		},
	}

	err := c.locatorController.Disable(ctx)
	if err != nil {
		c.logger.WithError(err).Error("failed to disable all locators")
		resp.GetAllLocatorsDisabled().Error = &v1.DaemonError{Error: err.Error()}
	}

	c.Send(resp)
}
