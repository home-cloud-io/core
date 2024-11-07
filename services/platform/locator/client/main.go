package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"connectrpc.com/connect"
	v1 "github.com/home-cloud-io/core/api/platform/locator/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/locator/v1/v1connect"
	"github.com/netbirdio/netbird/encryption"
	"golang.org/x/net/http2"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	privateKey = ""
	remotePublicKey = ""
	serverId = ""
)

func main() {
	ctx := context.Background()

	ourPrivateKey, err := wgtypes.ParseKey(privateKey)
	if err != nil {
		panic(err)
	}

	theirPublicKey, err := wgtypes.ParseKey(remotePublicKey)
	if err != nil {
		panic(err)
	}

	body, err := encryption.EncryptMessage(theirPublicKey, ourPrivateKey, &v1.LocateRequestBody{
		ServerId: serverId,
	})
	if err != nil {
		panic(err)
	}

	client := sdConnect.NewLocatorServiceClient(newInsecureClient(), "http://localhost:8001")
	res, err := client.Locate(ctx, &connect.Request[v1.LocateRequest]{
		Msg: &v1.LocateRequest{
			ServerId: serverId,
			Body: &v1.EncryptedMessage{
				PublicKey: ourPrivateKey.PublicKey().String(),
				Body: body,
			},
		},
	})
	if err != nil {
		fmt.Println("rejected")
		return
	}

	msg := &v1.LocateResponseBody{}
	err = encryption.DecryptMessage(theirPublicKey, ourPrivateKey, res.Msg.Body.Body, msg)
	if err != nil {
		panic(err)
	}
	fmt.Println(msg)
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