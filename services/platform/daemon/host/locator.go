package host

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/locator/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/locator/v1/v1connect"
	sv1 "github.com/home-cloud-io/core/api/platform/server/v1"
	"github.com/pion/stun/v2"

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
	LocatorController interface {
		// Load will load all saved Locators from blueprint and create background connections to them.
		// Meant to be called at service startup.
		Load()
		// AddLocator will start a background connection to the given Locator and will serve up connection
		// information to locate requests from that Locator on all interfaces. The Locator connection
		// can be killed by calling RemoveLocator or RemoveAll
		AddLocator(ctx context.Context, locatorAddress string) (locator *sv1.Locator, err error)
		// RemoveLocator will remove a background Locator connection that was started through Load or
		// AddLocator and will delete it from blueprint.
		RemoveLocator(ctx context.Context, locatorAddress string) error
		// Disable will remove all background Locator connections and delete them from blueprint.
		Disable(ctx context.Context) error
	}
	locatorController struct {
		logger      chassis.Logger
		stunClient  STUNClient
		stunAddress stun.XORMappedAddress
		// map of unique keys ("serverId@locatorAddress") to locator
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

func NewLocatorController(logger chassis.Logger, stun STUNClient) LocatorController {
	return &locatorController{
		logger:     logger,
		locators:   make(map[string]locator),
		stunClient: stun,
	}
}

func (m *locatorController) Load() {
	config := chassis.GetConfig()

	address, err := m.stunClient.Start()
	if err != nil {
		m.logger.WithError(err).Error("failed to get public address using STUN client")
		return
	}
	m.stunAddress = address

	// get settings from config
	settings := &sv1.LocatorSettings{}
	err = config.UnmarshalKey("daemon.locatorSettings", settings)
	if err != nil {
		m.logger.WithError(err).Warn("faied to get device settings when loading locators")
		return
	}

	// nothing to do if there are no locator settings
	if settings == nil || !settings.Enabled {
		return
	}

	for _, l := range settings.Locators {
		m.logger.WithFields(chassis.Fields{
			"locator_address": l.Address,
		}).Info("loading locator connection")

		for _, c := range l.Connections {
			// save locator information in memory
			ctx, cancel := context.WithCancel(context.Background())
			m.locators[locatorKey(l.Address, c.ServerId)] = locator{
				serverId: c.ServerId,
				address:  l.Address,
				cancel:   cancel,
			}

			// run connection in background (can be cancelled through context)
			go m.connectToLocator(ctx, m.logger, l.Address, c.ServerId, c.WireguardInterface)
		}

	}

}

func (m *locatorController) AddLocator(ctx context.Context, locatorAddress string) (*sv1.Locator, error) {
	config := chassis.GetConfig()

	// check if locator already exists in blueprint and reject if so
	settings := &sv1.LocatorSettings{}
	err := config.UnmarshalKey("daemon.locatorSettings", settings)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = &sv1.LocatorSettings{}
	}
	settings.Enabled = true
	if settings.Locators == nil {
		settings.Locators = make([]*sv1.Locator, 0)
	}
	for _, l := range settings.Locators {
		if l.Address == locatorAddress {
			return nil, fmt.Errorf("requested locator with the same interface name is already registered")
		}
	}

	// get the server's wireguard wgConfig
	wgConfig := &dv1.WireguardConfig{}
	err = config.UnmarshalKey("daemon.wireguard", wgConfig)
	if err != nil {
		m.logger.WithError(err).Error("failed to get wireguard config")
		return nil, err
	}

	connections := make([]*sv1.LocatorConnection, len(wgConfig.Interfaces))
	for index, inf := range wgConfig.Interfaces {
		// save locator information in memory
		ctx, cancel := context.WithCancel(context.Background())
		m.locators[locatorKey(locatorAddress, inf.Id)] = locator{
			serverId: inf.Id,
			address:  locatorAddress,
			cancel:   cancel,
		}

		// run connection in background (can be cancelled through context)
		go m.connectToLocator(ctx, m.logger, locatorAddress, inf.Id, inf.Name)

		connections[index] = &sv1.LocatorConnection{
			ServerId:           inf.Id,
			WireguardInterface: inf.Name,
		}
	}

	// save locator to config
	locator := &sv1.Locator{
		Address:     locatorAddress,
		Connections: connections,
	}
	settings.Locators = append(settings.Locators, locator)
	config.Set("daemon.locatorSettings", settings)
	err = config.WriteConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to save settings")
	}

	return locator, nil
}

func (m *locatorController) RemoveLocator(ctx context.Context, locatorAddress string) error {

	// filter out all locators to be deleted
	locators := make(map[string]locator)
	for key, l := range m.locators {
		// save those not matching
		if !strings.HasSuffix(key, locatorAddress) {
			locators[locatorKey(locatorAddress, l.serverId)] = l
			continue
		}
		// cancel those matching
		l.cancel()
	}
	m.locators = locators

	// TODO: delete from config
	// settings := &sv1.DeviceSettings{}
	// err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	// if err != nil {
	// 	return err
	// }
	// for i, l := range settings.LocatorSettings.Locators {
	// 	if l.Address == locatorAddress {
	// 		settings.LocatorSettings.Locators = append(settings.LocatorSettings.Locators[:i], settings.LocatorSettings.Locators[i+1:]...)
	// 	}
	// }
	// _, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	// if err != nil {
	// 	return fmt.Errorf("failed to save settings")
	// }

	return nil
}

func (m *locatorController) Disable(ctx context.Context) error {
	// TODO: this can be more efficient
	for _, l := range m.locators {
		err := m.RemoveLocator(ctx, l.address)
		if err != nil {
			return err
		}
	}

	// TODO: disable feature in config
	// settings := &sv1.DeviceSettings{}
	// err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	// if err != nil {
	// 	return err
	// }
	// settings.LocatorSettings = &sv1.LocatorSettings{
	// 	Enabled: false,
	// }
	// _, err = kvclient.Set(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	// if err != nil {
	// 	return fmt.Errorf("failed to save settings")
	// }

	return nil
}

func (m *locatorController) connectToLocator(ctx context.Context, logger chassis.Logger, locatorAddress, serverId, wgInterface string) {
	log := logger.WithFields(chassis.Fields{
		"locator_address": locatorAddress,
		"server_id":       serverId,
		"interface_name":  wgInterface,
	})
	log.Debug("connecting to locator")
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
			go m.authorizeLocate(ctx, log, wgInterface, stream, msg.GetLocate())
		default:
			log.WithError(err).Error("invalid message type received from locator")
		}
	}
}
func (m *locatorController) authorizeLocate(ctx context.Context, logger chassis.Logger, wgInterface string, stream *connect.BidiStreamForClient[v1.ServerMessage, v1.LocatorMessage], locate *v1.Locate) {
	config := chassis.GetConfig()

	// get wireguard config from blueprint
	wgConfig := &dv1.WireguardConfig{}
	err := config.UnmarshalKey("daemon.wireguard", wgConfig)
	if err != nil {
		m.logger.WithError(err).Error("failed to get wireguard config")
		reject(logger, locate.RequestId, stream)
		return
	}

	// make sure the wireguard interface exists (this is just protection if something desyncs and the interface is removed but this goroutine is not cancelled)
	var interfaceConfig *dv1.WireguardInterface
	for _, inf := range wgConfig.Interfaces {
		if inf.Name == wgInterface {
			interfaceConfig = inf
			break
		}
	}
	if interfaceConfig == nil {
		logger.WithError(fmt.Errorf("wireguard interface [%s] does not exist in config", wgInterface)).Error("failed to get wireguard interface from config")
		reject(logger, locate.RequestId, stream)
		return
	}

	// convert server settings to internal wireguard types
	iConfig, err := parseConfig(interfaceConfig.Name, interfaceConfig.Id)
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
	authorized, err := validate(logger, iConfig, remoteKey, locate.Body.Body)
	if authorized && err == nil {
		m.accept(logger, iConfig, remoteKey, stream, locate)
		return
	}
	if err != nil {
		logger.WithError(err).Error("failed to validate locate request")
	}
	reject(log, locate.RequestId, stream)
}

func validate(logger chassis.Logger, config WireGuardConfig, remoteKey wgtypes.Key, body []byte) (authorized bool, err error) {
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

			logger.WithFields(chassis.Fields{
				"requested": msg.ServerId,
				"actual":    config.Id,
			}).Debug("server id does not match")
			return false, nil
		}
	}
	logger.Debug("given public key not in trusted peers")
	return false, nil
}

func (m *locatorController) accept(logger chassis.Logger, config WireGuardConfig, remoteKey wgtypes.Key, stream *connect.BidiStreamForClient[v1.ServerMessage, v1.LocatorMessage], locate *v1.Locate) {
	logger.Info("approving request")

	msg := &v1.LocateResponseBody{
		Address: m.stunAddress.IP.String(),
		Port:    uint32(m.stunAddress.Port),
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
		reject(logger, locate.RequestId, stream)
		return
	}

	// attempt outbound connection to peer to open hole in NAT
	peerAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", msg.Address, msg.Port))
	if err != nil {
		logger.WithError(err).Error("failed to resolve UDP address")
		return
	}
	m.stunClient.Connect(peerAddr)
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
func parseConfig(name, id string) (config WireGuardConfig, err error) {

	// read config
	netConfig := NetworkingConfig{}
	f, err := os.ReadFile(NetworkingConfigFile())
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(f, &netConfig)
	if err != nil {
		return config, err
	}

	data, err := os.ReadFile(netConfig.Wireguard.Interfaces[name].PrivateKeyFile)
	if err != nil {
		return config, err
	}

	ourPrivateKey, err := wgtypes.ParseKey(string(data))
	if err != nil {
		return config, err
	}
	config.PrivateKey = ourPrivateKey
	config.PublicKey = ourPrivateKey.PublicKey()

	peers := netConfig.Wireguard.Interfaces[name].Peers
	config.Peers = make([]wgtypes.Key, len(peers))
	for index, peer := range peers {
		publicKey, err := wgtypes.ParseKey(peer.PublicKey)
		if err != nil {
			return config, err
		}
		config.Peers[index] = publicKey
	}

	config.Id = id
	return config, nil
}

func locatorKey(locatorAddress, serverId string) string {
	return fmt.Sprintf("%s@%s", serverId, locatorAddress)
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
