package locator

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"connectrpc.com/connect"
	v1 "github.com/home-cloud-io/core/api/platform/locator/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/locator/v1/v1connect"

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
)

const (
	fakeAccessToken = "fake_access_token"
)

func Connect(ctx context.Context, logger chassis.Logger, address string) {
	config, err := parseConfig()
	if err != nil {
		panic(err)
	}

	client := sdConnect.NewLocatorServiceClient(newInsecureClient(), address)
	stream := client.Connect(ctx)

	err = stream.Send(&v1.ServerMessage{
		AccessToken: fakeAccessToken,
		Body: &v1.ServerMessage_Initialize{
			Initialize: &v1.Initialize{
				ServerId: config.Id,
			},
		},
	})
	if err != nil {
		panic(err)
	}

	for {
		msg, err := stream.Receive()
		if err != nil {
			panic(err)
		}
		switch msg.Body.(type) {
		case *v1.LocatorMessage_Locate:
			go authorizeLocate(logger, config, stream, msg.GetLocate())
		default:
			panic("invalid type")
		}
	}
}
func authorizeLocate(logger chassis.Logger, config WireGuardConfig, stream *connect.BidiStreamForClient[v1.ServerMessage, v1.LocatorMessage], locate *v1.Locate) {
	log := logger.WithField("peer_public_key", locate.Body.PublicKey)
	log.Info("received locator connect request")
	remoteKey, err := wgtypes.ParseKey(locate.Body.PublicKey)
	if err != nil {
		reject(log, locate.RequestId, stream)
		return
	}

	authorized, err := validate(config, remoteKey, locate.Body.Body)
	if authorized && err == nil {
		log.Info("approving request")
		msg := &v1.LocateResponseBody{
			Address: "localhost",
			Port:    8000,
		}
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
			reject(log, locate.RequestId, stream)
		}
		return
	}

	reject(log, locate.RequestId, stream)
}

func validate(config WireGuardConfig, remoteKey wgtypes.Key, body []byte) (authorized bool, err error) {
	for _, trustedKey := range config.Peers {
		if trustedKey == remoteKey {
			// decrypt message
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
	logger.Warn("rejecting request")
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

func parseConfig() (config WireGuardConfig, err error) {
	c := chassis.GetConfig()
	config.Id = c.GetString("server.wireguard.id")

	ourPrivateKey, err := wgtypes.ParseKey(c.GetString("server.wireguard.private_key"))
	if err != nil {
		return config, err
	}
	config.PrivateKey = ourPrivateKey
	config.PublicKey = ourPrivateKey.PublicKey()

	peers := c.GetStringSlice("server.wireguard.peers")
	config.Peers = make([]wgtypes.Key, len(peers))
	for index, peer := range peers {
		publicKey, err := wgtypes.ParseKey(peer)
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
