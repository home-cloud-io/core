package host

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	v1 "github.com/home-cloud-io/core/api/platform/locator/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/locator/v1/v1connect"
	sv1 "github.com/home-cloud-io/core/api/platform/server/v1"
	"github.com/home-cloud-io/core/services/platform/daemon/host/encryption"

	"connectrpc.com/connect"
	"github.com/steady-bytes/draft/pkg/chassis"
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
		Connect(ctx context.Context, wgInterface *sv1.WireguardInterface, locatorAddress string) error
		Close(wgInterface *sv1.WireguardInterface, locatorAddress string)
	}
	locatorController struct {
		logger         chassis.Logger
		stunController STUNController
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

func NewLocatorController(logger chassis.Logger, stunController STUNController) LocatorController {
	return &locatorController{
		logger:         logger,
		locators:       make(map[string]locator),
		stunController: stunController,
	}
}

func (m *locatorController) Connect(ctx context.Context, wgInterface *sv1.WireguardInterface, locatorAddress string) error {
	// save locator information in memory
	ctx, cancel := context.WithCancel(context.Background())
	m.locators[locatorKey(locatorAddress, wgInterface.Id)] = locator{
		serverId: wgInterface.Id,
		address:  locatorAddress,
		cancel:   cancel,
	}

	// run connection in background (can be cancelled through context)
	go m.connectToLocator(ctx, m.logger, wgInterface, locatorAddress)

	return nil
}

func (m *locatorController) Close(wgInterface *sv1.WireguardInterface, locatorAddress string) {
	locator, ok := m.locators[locatorKey(locatorAddress, wgInterface.Id)]
	if !ok {
		return
	}
	locator.cancel()
	delete(m.locators, locatorKey(locatorAddress, wgInterface.Id))
}

func (m *locatorController) connectToLocator(ctx context.Context, logger chassis.Logger, wgInterface *sv1.WireguardInterface, locatorAddress string) {
	log := logger.WithFields(chassis.Fields{
		"locator_address": locatorAddress,
		"server_id":       wgInterface.Id,
		"interface_name":  wgInterface.Name,
	})
	log.Debug("connecting to locator")

	client := sdConnect.NewLocatorServiceClient(http.DefaultClient, locatorAddress)
	stream := client.Connect(ctx)

	err := stream.Send(&v1.ServerMessage{
		AccessToken: fakeAccessToken,
		Body: &v1.ServerMessage_Initialize{
			Initialize: &v1.Initialize{
				ServerId: wgInterface.Id,
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
func (m *locatorController) authorizeLocate(ctx context.Context, logger chassis.Logger, wgInterface *sv1.WireguardInterface, stream *connect.BidiStreamForClient[v1.ServerMessage, v1.LocatorMessage], locate *v1.Locate) {
	// convert server settings to internal wireguard types
	wgConfig, err := parseConfig(wgInterface.Name, wgInterface.Id)
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
	authorized, request, err := validate(logger, wgConfig, remoteKey, locate.Body.Body)
	if authorized && err == nil {
		m.accept(logger, wgInterface, wgConfig, remoteKey, stream, locate, request)
		return
	}
	if err != nil {
		logger.WithError(err).Error("failed to validate locate request")
	}
	reject(log, locate.RequestId, stream)
}

func validate(logger chassis.Logger, config WireGuardConfig, remoteKey wgtypes.Key, body []byte) (authorized bool, request *v1.LocateRequestBody, err error) {
	for _, trustedKey := range config.Peers {
		if trustedKey == remoteKey {
			// attempt to decrypt message using our private key and their given public key
			request = &v1.LocateRequestBody{}
			err = encryption.DecryptMessage(remoteKey, config.PrivateKey, body, request)
			if err != nil {
				return false, nil, err
			}

			// validate the encrypted server id matches our own
			if request.ServerId == config.Id {
				return true, request, nil
			}

			logger.WithFields(chassis.Fields{
				"requested": request.ServerId,
				"actual":    config.Id,
			}).Debug("server id does not match")
			return false, nil, nil
		}
	}
	logger.Debug("given public key not in trusted peers")
	return false, nil, nil
}

func (m *locatorController) accept(logger chassis.Logger, wgInterface *sv1.WireguardInterface, config WireGuardConfig, remoteKey wgtypes.Key, stream *connect.BidiStreamForClient[v1.ServerMessage, v1.LocatorMessage], locate *v1.Locate, request *v1.LocateRequestBody) {
	logger.Info("approving request")

	// get the current address from the STUN binding
	address, err := m.stunController.Address(int(wgInterface.Port))
	if err != nil {
		logger.WithError(err).Error("failed to get STUN address")
		reject(logger, locate.RequestId, stream)
		return
	}

	msg := &v1.LocateResponseBody{
		Address: address.IP.String(),
		Port:    uint32(address.Port),
	}

	// encrypt the response before sending
	body, err := encryption.EncryptMessage(remoteKey, config.PrivateKey, msg)
	if err != nil {
		logger.WithError(err).Error("failed to encrypt accept message")
		reject(logger, locate.RequestId, stream)
		return
	}

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
	logger.WithField("address", fmt.Sprintf("%s:%d", request.Address, request.Port)).Debug("attempting peer connection")
	peerAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", request.Address, request.Port))
	if err != nil {
		logger.WithError(err).Error("failed to resolve UDP address")
		return
	}
	m.stunController.Connect(int(wgInterface.Port), peerAddr)
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

	// TODO: read from blueprint
	netConfig := NetworkingConfig{}

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
