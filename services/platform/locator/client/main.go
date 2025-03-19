package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	v1 "github.com/home-cloud-io/core/api/platform/locator/v1"
	sdConnect "github.com/home-cloud-io/core/api/platform/locator/v1/v1connect"

	"connectrpc.com/connect"
	"github.com/netbirdio/netbird/encryption"
	"github.com/pion/stun/v2"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	privateKey      = ""
	remotePublicKey = "DgsKlPzywIdx5e0039H+HgRibsGZvcNCIq90sHpzITw="
	serverId        = "4a306461-e3fb-4b8c-a5f0-9052370fddcc"

	stunServer    = "locator1.home-cloud.io:3478"
	locatorServer = "https://locator1.home-cloud.io"
)

func copyAddr(dst *stun.XORMappedAddress, src stun.XORMappedAddress) {
	dst.IP = append(dst.IP, src.IP...)
	dst.Port = src.Port
}

func keepAlive(c *stun.Client) {
	// Keep-alive for NAT binding.
	t := time.NewTicker(time.Second * 5)
	for range t.C {
		if err := c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
			if res.Error != nil {
				log.Panicf("Failed STUN transaction: %s", res.Error)
			}
		}); err != nil {
			log.Panicf("Failed STUN transaction: %s", err)
		}
	}
}

type message struct {
	text string
	addr net.Addr
}

func demultiplex(conn *net.UDPConn, stunConn io.Writer, messages chan message) {
	buf := make([]byte, 1024)
	for {
		n, raddr, err := conn.ReadFrom(buf)
		if err != nil {
			log.Panicf("Failed to read: %s", err)
		}

		// De-multiplexing incoming packets.
		if stun.IsMessage(buf[:n]) {
			// If buf looks like STUN message, send it to STUN client connection.
			if _, err = stunConn.Write(buf[:n]); err != nil {
				log.Panicf("Failed to write: %s", err)
			}
		} else {
			// If not, it is application data.
			log.Printf("Demultiplex: [%s]: %s", raddr, buf[:n])
			messages <- message{
				text: string(buf[:n]),
				addr: raddr,
			}
		}
	}
}

func multiplex(conn *net.UDPConn, stunAddr net.Addr, stunConn io.Reader) {
	// Sending all data from stun client to stun server.
	buf := make([]byte, 1024)
	for {
		n, err := stunConn.Read(buf)
		if err != nil {
			log.Panicf("Failed to read: %s", err)
		}
		if _, err = conn.WriteTo(buf[:n], stunAddr); err != nil {
			log.Panicf("Failed to write: %s", err)
		}
	}
}

func main() {
	ctx := context.Background()

	// Allocating local UDP socket that will be used both for STUN and
	// our application data.
	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:0")
	if err != nil {
		log.Panicf("Failed to resolve: %s", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Panicf("Failed to listen: %s", err)
	}

	// Resolving STUN server address.
	stunAddr, err := net.ResolveUDPAddr("udp4", stunServer)
	if err != nil {
		log.Panicf("Failed to resolve '%s': %s", stunServer, err)
	}

	stunL, stunR := net.Pipe()

	c, err := stun.NewClient(stunR)
	if err != nil {
		log.Panicf("Failed to create client: %s", err)
	}

	// Starting multiplexing (writing back STUN messages) with de-multiplexing
	// (passing STUN messages to STUN client and processing application
	// data separately).
	//
	// stunL and stunR are virtual connections, see net.Pipe for reference.
	messages := make(chan message)

	go demultiplex(conn, stunL, messages)
	go multiplex(conn, stunAddr, stunL)

	// Getting our "real" IP address from STUN Server.
	// This will create a NAT binding on your provider/router NAT Server,
	// and the STUN server will return allocated public IP for that binding.
	//
	// This can fail if your NAT Server is strict and will use separate ports
	// for application data and STUN
	var gotAddr stun.XORMappedAddress
	if err = c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			log.Panicf("Failed STUN transaction: %s", res.Error)
		}
		var xorAddr stun.XORMappedAddress
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			log.Panicf("Failed to get XOR-MAPPED-ADDRESS: %s", getErr)
		}
		copyAddr(&gotAddr, xorAddr)
	}); err != nil {
		log.Panicf("Failed STUN transaction: %s", err)
	}

	log.Printf("Public address: %s", gotAddr)

	// Keep-alive is needed to keep our NAT port allocated.
	// Any ping-pong will work, but we are just making binding requests.
	// Note that STUN Server is not mandatory for keep alive, application
	// data will keep alive that binding too.
	go keepAlive(c)

	go func() {
		for m := range messages {
			// TODO: just print messages for now
			log.Println(m.text)
		}
	}()

	// communicate with locator

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
		Address:  gotAddr.IP.String(),
		Port:     uint32(gotAddr.Port),
	})
	if err != nil {
		panic(err)
	}

	client := sdConnect.NewLocatorServiceClient(http.DefaultClient, locatorServer)
	res, err := client.Locate(ctx, &connect.Request[v1.LocateRequest]{
		Msg: &v1.LocateRequest{
			ServerId: serverId,
			Body: &v1.EncryptedMessage{
				PublicKey: ourPrivateKey.PublicKey().String(),
				Body:      body,
			},
		},
	})
	if err != nil {
		fmt.Printf("rejected: %s", err)
		return
	}

	msg := &v1.LocateResponseBody{}
	err = encryption.DecryptMessage(theirPublicKey, ourPrivateKey, res.Msg.Body.Body, msg)
	if err != nil {
		panic(err)
	}
	fmt.Println("Response from locator: ", msg)

	peerAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", msg.Address, msg.Port))
	if err != nil {
		log.Panicf("Failed to resolve '%s': %s", fmt.Sprintf("%s:%d", msg.Address, msg.Port), err)
	}

	deadline := time.After(time.Second * 10)

	sendMsg := func() {
		msg := uuid.New().String()
		log.Printf("sending: %s", msg)
		if _, err := conn.WriteTo([]byte(msg), peerAddr); err != nil {
			log.Panicf("Failed to write: %s", err)
		}
	}

	for {
		select {
		case <-deadline:
			log.Print("finished attempt to open connection to peer")
			return
		case <-time.After(time.Second):
			sendMsg()
		}
	}

}
