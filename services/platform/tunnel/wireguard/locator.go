package wireguard

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	lv1 "github.com/home-cloud-io/core/api/platform/locator/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/locator/v1/v1connect"
	"github.com/home-cloud-io/core/services/platform/daemon/host/encryption"
	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/services/platform/tunnel/stun"
)

type LocatorClient struct {
	Address            string
	WireguardReference types.NamespacedName
	KubeClient         client.Client
	STUNController     stun.STUNController
	cancel             context.CancelFunc
}

const (
	fakeAccessToken = "fake_access_token"
)

func (l *LocatorClient) Connect(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	l.cancel = cancel

	log := log.FromContext(ctx)
	log.V(1).Info("connecting to locator")

	client := sdConnect.NewLocatorServiceClient(http.DefaultClient, l.Address)
	stream := client.Connect(ctx)

	// get wireguard interface from k8s
	wg := v1.Wireguard{}
	err := l.KubeClient.Get(ctx, l.WireguardReference, &wg)
	if err != nil {
		log.Error(err, "failed to get Wireguard object from k8s")
		return
	}

	// start stream to locator server
	err = stream.Send(&lv1.ServerMessage{
		AccessToken: fakeAccessToken,
		Body: &lv1.ServerMessage_Initialize{
			Initialize: &lv1.Initialize{
				ServerId: wg.Spec.ID,
			},
		},
	})
	if err != nil {
		log.Error(err, "failed to initialize stream with locator")
		return
	}

	for {
		if err := ctx.Err(); err != nil {
			log.Info("connection to locator was closed", "reason", err)
			return
		}

		msg, err := stream.Receive()
		if err != nil {
			if !strings.Contains(err.Error(), "context canceled") {
				log.Error(err, "failed to receive message from locator")
			}
			continue
		}
		switch msg.Body.(type) {
		case *lv1.LocatorMessage_Locate:
			go l.authorizeLocate(ctx, stream, msg.GetLocate())
		default:
			log.Error(err, "invalid message type received from locator")
		}
	}
}

func (l *LocatorClient) Cancel() {
	l.cancel()
}

func (l *LocatorClient) authorizeLocate(ctx context.Context, stream *connect.BidiStreamForClient[lv1.ServerMessage, lv1.LocatorMessage], locate *lv1.Locate) {
	log := log.FromContext(ctx)
	log.Info("received locator connect request")
	remoteKey, err := wgtypes.ParseKey(locate.Body.PublicKey)
	if err != nil {
		log.Error(err, "failed to parse public key from message body")
		reject(ctx, locate.RequestId, stream)
		return
	}

	// get the wireguard configuration from k8s to validate the request
	// we want to pull it from k8s on every request so that we don't use stale configuration which
	// could lead to authorizing a connection for a peer that has been revoked by the user

	// get wireguard interface from k8s
	wg := v1.Wireguard{}
	err = l.KubeClient.Get(ctx, l.WireguardReference, &wg)
	if err != nil {
		log.Error(err, "failed to get Wireguard object from k8s")
		reject(ctx, locate.RequestId, stream)
		return
	}

	// build wireguard config
	config, err := buildConfig(ctx, l.KubeClient, wg)
	if err != nil {
		log.Error(err, "failed to build Wireguard config from kube object")
		reject(ctx, locate.RequestId, stream)
		return
	}

	// attempt to validate the locate request and reject if we can't validate it
	authorized, request, err := l.validate(ctx, wg.Spec.ID, config, remoteKey, locate.Body.Body)
	if authorized && err == nil {
		l.accept(ctx, config, remoteKey, stream, locate, request)
		return
	}
	if err != nil {
		log.Error(err, "failed to validate locate request")
	}
	reject(ctx, locate.RequestId, stream)
}

func (l *LocatorClient) validate(ctx context.Context, serverID string, config wgtypes.Config, remoteKey wgtypes.Key, body []byte) (authorized bool, request *lv1.LocateRequestBody, err error) {
	log := log.FromContext(ctx)
	for _, peer := range config.Peers {
		if peer.PublicKey == remoteKey {

			// attempt to decrypt message using our private key and their given public key
			request = &lv1.LocateRequestBody{}
			err = encryption.DecryptMessage(remoteKey, *config.PrivateKey, body, request)
			if err != nil {
				return false, nil, err
			}

			// validate the encrypted server id matches our own
			if request.ServerId == serverID {
				return true, request, nil
			}

			log.V(1).Info("server id does not match")
			return false, nil, nil
		}
	}
	log.V(1).Info("given public key not in trusted peers")
	return false, nil, nil
}

func (l *LocatorClient) accept(ctx context.Context, config wgtypes.Config, remoteKey wgtypes.Key, stream *connect.BidiStreamForClient[lv1.ServerMessage, lv1.LocatorMessage], locate *lv1.Locate, request *lv1.LocateRequestBody) {
	log := log.FromContext(ctx)
	log.Info("approving request")

	// get the current address from the STUN binding
	address, err := l.STUNController.Address(*config.ListenPort)
	if err != nil {
		log.Error(err, "failed to get STUN address")
		reject(ctx, locate.RequestId, stream)
		return
	}

	msg := &lv1.LocateResponseBody{
		Address: address.IP.String(),
		Port:    uint32(address.Port),
	}

	// encrypt the response before sending
	body, err := encryption.EncryptMessage(remoteKey, *config.PrivateKey, msg)
	if err != nil {
		log.Error(err, "failed to encrypt accept message")
		reject(ctx, locate.RequestId, stream)
		return
	}

	err = stream.Send(&lv1.ServerMessage{
		AccessToken: fakeAccessToken,
		Body: &lv1.ServerMessage_Accept{
			Accept: &lv1.Accept{
				RequestId: locate.RequestId,
				Body: &lv1.EncryptedMessage{
					PublicKey: config.PrivateKey.PublicKey().String(),
					Body:      body,
				},
			},
		},
	})
	if err != nil {
		log.Error(err, "failed to send accept message")
		reject(ctx, locate.RequestId, stream)
		return
	}

	// attempt outbound connection to peer to open hole in NAT
	log.V(1).Info("attempting peer connection")
	peerAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", request.Address, request.Port))
	if err != nil {
		log.Error(err, "failed to resolve UDP address")
		return
	}
	l.STUNController.Connect(*config.ListenPort, peerAddr)
}

func reject(ctx context.Context, requestId string, stream *connect.BidiStreamForClient[lv1.ServerMessage, lv1.LocatorMessage]) {
	log := log.FromContext(ctx)
	log.GetSink().Info(-1, "rejecting locate request")
	err := stream.Send(&lv1.ServerMessage{
		AccessToken: fakeAccessToken,
		Body: &lv1.ServerMessage_Reject{
			Reject: &lv1.Reject{
				RequestId: requestId,
			},
		},
	})
	if err != nil {
		log.Error(err, "failed to send rejection message")
	}
}
