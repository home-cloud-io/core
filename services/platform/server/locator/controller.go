package locator

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/locator/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/locator/v1/v1connect"
	sv1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"

	"connectrpc.com/connect"
	"github.com/netbirdio/netbird/encryption"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/net/http2"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type (
	WireGuardConfig struct {
		Id         string
		PrivateKey wgtypes.Key
		PublicKey  wgtypes.Key
		Peers      []wgtypes.Key
	}
	Controller interface {
		// Load will load all saved Locators from blueprint and create background connections to them.
		// Meant to be called at service startup.
		Load()
		// AddLocator will start a background connection to the given Locator and will serve up connection
		// information to locate requests from that Locator on the given interface. The Locator connection
		// can be killed by calling RemoveLocator or RemoveAll
		AddLocator(ctx context.Context, locatorAddress string, wgInterface string) error
		// RemoveLocator will remove a background Locator connection that was started through Load or
		// AddLocator and will delete it from blueprint.
		RemoveLocator(ctx context.Context, serverId string) error
		// Disable will remove all background Locator connections and delete them from blueprint.
		Disable(ctx context.Context) error
	}
	controller struct {
		logger chassis.Logger
		// map of unique serverId to locator
		locators map[string]locator
	}
	locator struct {
		serverId string
		address  string
		cancel   context.CancelFunc
	}
)

const (
	fakeAccessToken = "fake_access_token"
)

func NewController(logger chassis.Logger) Controller {
	return &controller{
		logger:   logger,
		locators: make(map[string]locator),
	}
}

func (m *controller) Load() {
	ctx := context.Background()

	// get settings from blueprint
	settings := &sv1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		m.logger.WithError(err).Warn("faied to get device settings when loading locators")
		return
	}

	// nothing to do if there are no locator settings
	if settings.LocatorSettings == nil {
		return
	}

	for _, l := range settings.LocatorSettings.Locators {
		m.logger.WithFields(chassis.Fields{
			"server_id":           l.ServerId,
			"locator_address":     l.Address,
			"wireguard_interface": l.WireguardInterface,
		}).Info("loading locator connection")

		// save locator information in memory
		ctx, cancel := context.WithCancel(context.Background())
		m.locators[l.ServerId] = locator{
			serverId: l.ServerId,
			address:  l.Address,
			cancel:   cancel,
		}

		// run connection in background (can be cancelled through context)
		go connectToLocator(ctx, m.logger, l.Address, l.ServerId, l.WireguardInterface)
	}

}

func (m *controller) AddLocator(ctx context.Context, locatorAddress string, wgInterface string) error {

	// check if locator already exists in blueprint and reject if so
	settings := &sv1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return err
	}
	settings.LocatorSettings.Enabled = true
	if settings.LocatorSettings.Locators == nil {
		settings.LocatorSettings.Locators = make(map[string]*sv1.Locator)
	}
	for _, l := range settings.LocatorSettings.Locators {
		if l.Address == locatorAddress && l.WireguardInterface == wgInterface {
			return fmt.Errorf("requested locator with the same interface name is already registered")
		}
	}

	// get the server's wireguard wgConfig
	wgConfig := &dv1.WireguardConfig{}
	err = kvclient.Get(ctx, kvclient.WIREGUARD_CONFIG_KEY, wgConfig)
	if err != nil {
		m.logger.WithError(err).Error("failed to get wireguard config")
		return err
	}

	// make sure the underlying wireguard interface exists
	inf, ok := wgConfig.Interfaces[wgInterface]
	if !ok {
		return fmt.Errorf("failed to get wireguard interface from config")
	}

	// make sure the locator connection doesn't already exist
	_, ok = m.locators[inf.Id]
	if ok {
		return fmt.Errorf("the requested locator address is already in use")
	}

	// save locator information in memory
	ctx, cancel := context.WithCancel(context.Background())
	m.locators[inf.Id] = locator{
		serverId: inf.Id,
		address:  locatorAddress,
		cancel:   cancel,
	}

	// run connection in background (can be cancelled through context)
	go connectToLocator(ctx, m.logger, locatorAddress, inf.Id, inf.Name)

	// save locator to blueprint
	settings.LocatorSettings.Locators[inf.Id] = &sv1.Locator{
		ServerId:           inf.Id,
		Address:            locatorAddress,
		WireguardInterface: inf.Name,
	}
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return fmt.Errorf("failed to save settings")
	}

	return nil
}

func (m *controller) RemoveLocator(ctx context.Context, serverId string) error {
	l, ok := m.locators[serverId]
	if !ok {
		return fmt.Errorf("locator not found in current connections")
	}
	l.cancel()
	delete(m.locators, serverId)

	// delete from blueprint
	settings := &sv1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return err
	}
	delete(settings.LocatorSettings.Locators, serverId)
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return fmt.Errorf("failed to save settings")
	}

	return nil
}

func (m *controller) Disable(ctx context.Context) error {
	for serverId, _ := range m.locators {
		err := m.RemoveLocator(ctx, serverId)
		if err != nil {
			return err
		}
	}

	// disable feature in blueprint
	settings := &sv1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return err
	}
	settings.LocatorSettings.Enabled = false
	_, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		return fmt.Errorf("failed to save settings")
	}

	return nil
}

func connectToLocator(ctx context.Context, logger chassis.Logger, locatorAddress, serverId, wgInterface string) {
	log := logger.WithFields(chassis.Fields{
		"locator_address": locatorAddress,
		"server_id":       serverId,
		"interface_name":  wgInterface,
	})
	client := sdConnect.NewLocatorServiceClient(newInsecureClient(), locatorAddress)
	stream := client.Connect(ctx)

	err := stream.Send(&v1.ServerMessage{
		AccessToken: fakeAccessToken,
		Body: &v1.ServerMessage_Initialize{
			Initialize: &v1.Initialize{
				ServerId: serverId,
			},
		},
	})
	if err != nil {
		log.WithError(err).Error("failed to initialize stream with locator")
		return
	}

	for {
		msg, err := stream.Receive()
		if err != nil {
			if !strings.Contains(err.Error(), "context canceled") {
				log.WithError(err).Error("failed to receive message from locator")
			}
			return
		}
		switch msg.Body.(type) {
		case *v1.LocatorMessage_Locate:
			go authorizeLocate(ctx, log, wgInterface, stream, msg.GetLocate())
		default:
			log.WithError(err).Error("invalid message type received from locator")
		}
	}
}
func authorizeLocate(ctx context.Context, logger chassis.Logger, wgInterface string, stream *connect.BidiStreamForClient[v1.ServerMessage, v1.LocatorMessage], locate *v1.Locate) {

	// get wireguard config from blueprint
	c := &dv1.WireguardConfig{}
	err := kvclient.Get(ctx, kvclient.WIREGUARD_CONFIG_KEY, c)
	if err != nil {
		logger.WithError(err).Error("failed to get wireguard config")
		reject(logger, locate.RequestId, stream)
		return
	}

	// make sure the wireguard interface exists (this is just protection if something desyncs and the interface is removed but this goroutine is not cancelled)
	interfaceConfig, ok := c.Interfaces[wgInterface]
	if !ok {
		logger.WithError(fmt.Errorf("wireguard interface [%s] does not exist in config")).Error("failed to get wireguard interface from config")
		reject(logger, locate.RequestId, stream)
		return
	}

	// convert server settings to internal wireguard types
	config, err := parseConfig(interfaceConfig)
	if err != nil {
		logger.WithError(err).Error("failed to parse wireguard config")
		reject(logger, locate.RequestId, stream)
		return
	}

	log := logger.WithField("peer_public_key", locate.Body.PublicKey)
	log.Info("received locator connect request")
	remoteKey, err := wgtypes.ParseKey(locate.Body.PublicKey)
	if err != nil {
		logger.WithError(err).Error("failed to parse public key from message body")
		reject(log, locate.RequestId, stream)
		return
	}

	// attempt to validate the locate request and reject if we can't validate it
	authorized, err := validate(config, remoteKey, locate.Body.Body)
	if authorized && err == nil {
		log.Info("approving request")
		// TODO: return STUN information
		msg := &v1.LocateResponseBody{
			Address: "localhost",
			Port:    8000,
		}
		// encrypt the response before sending
		body, err := encryption.EncryptMessage(remoteKey, config.PrivateKey, msg)
		err = stream.Send(&v1.ServerMessage{
			AccessToken: fakeAccessToken,
			Body: &v1.ServerMessage_Accept{
				Accept: &v1.Accept{
					RequestId: locate.RequestId,
					Body: &v1.EncryptedMessage{
						PublicKey: config.PublicKey.String(),
						Body:      body,
					},
				},
			},
		})
		if err != nil {
			logger.WithError(err).Error("failed to send accept message")
			reject(log, locate.RequestId, stream)
			return
		}
		return
	}
	reject(log, locate.RequestId, stream)
}

func validate(config WireGuardConfig, remoteKey wgtypes.Key, body []byte) (authorized bool, err error) {
	for _, trustedKey := range config.Peers {
		if trustedKey == remoteKey {
			// attempt to decrypt message using our private key and their given public key
			msg := &v1.LocateRequestBody{}
			err = encryption.DecryptMessage(remoteKey, config.PrivateKey, body, msg)
			if err != nil {
				return false, err
			}

			// validate the encrypted server id matches our own
			if msg.ServerId == config.Id {
				return true, nil
			}

			return false, nil
		}
	}
	return false, nil
}

func reject(logger chassis.Logger, requestId string, stream *connect.BidiStreamForClient[v1.ServerMessage, v1.LocatorMessage]) {
	logger.Warn("rejecting locate request")
	err := stream.Send(&v1.ServerMessage{
		AccessToken: fakeAccessToken,
		Body: &v1.ServerMessage_Reject{
			Reject: &v1.Reject{
				RequestId: requestId,
			},
		},
	})
	if err != nil {
		logger.WithError(err).Error("failed to send rejection message")
	}
}

// parseConfig takes the wireguard interface from the server's settings and converts everything to internal
// types (like wgtypes.Key instead of string) for encryption/decryption and peer validation
func parseConfig(c *dv1.WireguardInterface) (config WireGuardConfig, err error) {
	ourPrivateKey, err := wgtypes.ParseKey(c.PrivateKey)
	if err != nil {
		return config, err
	}
	config.PrivateKey = ourPrivateKey
	config.PublicKey = ourPrivateKey.PublicKey()

	peers := c.Peers
	config.Peers = make([]wgtypes.Key, len(peers))
	for index, peer := range peers {
		publicKey, err := wgtypes.ParseKey(peer.PublicKey)
		if err != nil {
			return config, err
		}
		config.Peers[index] = publicKey
	}

	return config, nil
}

// TODO: replace with secure client and only use this one when running locally during development
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
