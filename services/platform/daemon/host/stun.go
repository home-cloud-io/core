package host

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/pion/stun/v2"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	STUNClient interface {
		Start() (address stun.XORMappedAddress, err error)
		Bind(server string) (stun.XORMappedAddress, error)
		Connect(address net.Addr)
	}
	stunClient struct {
		logger chassis.Logger
		client *stun.Client
		conn   *net.UDPConn
	}
)

func NewSTUNClient(logger chassis.Logger) STUNClient {
	return &stunClient{
		logger: logger,
	}
}

func (c *stunClient) Start() (address stun.XORMappedAddress, err error) {

	if c.client != nil {
		return
	}

	config := chassis.GetConfig()

	server := config.GetString("daemon.locatorSettings.stunServerAddress")
	if server == "" {
		msg := "no stun server defined in the config"
		c.logger.Error(msg)
		return address, fmt.Errorf(msg)
	}

	return c.bind(c.logger, server)
}

func (c *stunClient) Bind(server string) (address stun.XORMappedAddress, err error) {

	if c.client != nil {
		err := c.client.Close()
		if err != nil {
			c.logger.WithError(err).Error("failed to close client")
			return address, err
		}
	}

	return c.bind(c.logger, server)
}

func copyAddr(dst *stun.XORMappedAddress, src stun.XORMappedAddress) {
	dst.IP = append(dst.IP, src.IP...)
	dst.Port = src.Port
}

func (c *stunClient) Connect(address net.Addr) {
	deadline := time.After(time.Second * 10)

	sendMsg := func() {
		msg := uuid.New().String()
		if _, err := c.conn.WriteTo([]byte(msg), address); err != nil {
			log.Panicf("Failed to write: %s", err)
		}
	}

	for {
		select {
		case <-deadline:
			c.logger.Debug("finished attempt to open connection to peer")
		case <-time.After(time.Second):
			// Retry
			sendMsg()
			// case m := <-messages:
			// 	log.Printf("Got response from %s: %s", m.addr, m.text)
			// case <-notify:
			// 	log.Print("Stopping")
			// 	return
		}
	}
}

func keepAlive(logger chassis.Logger, c *stun.Client) {
	// Keep-alive for NAT binding.
	t := time.NewTicker(time.Second * 5)
	for range t.C {
		if err := c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
			if res.Error != nil {
				logger.WithError(res.Error).Error("failed STUN transaction")
				return
			}
		}); err != nil {
			logger.WithError(err).Error("failed STUN transaction")
			return
		}
	}
}

type message struct {
	text string
	addr net.Addr
}

func demultiplex(logger chassis.Logger, conn *net.UDPConn, stunConn io.Writer, messages chan message) {
	buf := make([]byte, 1024)
	for {
		n, raddr, err := conn.ReadFrom(buf)
		if err != nil {
			logger.WithError(err).Error("failed to read")
			return
		}

		// De-multiplexing incoming packets.
		if stun.IsMessage(buf[:n]) {
			// If buf looks like STUN message, send it to STUN client connection.
			if _, err = stunConn.Write(buf[:n]); err != nil {
				logger.WithError(err).Error("failed to write")
				return
			}
		} else {
			// If not, it is application data.
			logger.Infof("Demultiplex: [%s]: %s", raddr, buf[:n])
			messages <- message{
				text: string(buf[:n]),
				addr: raddr,
			}
		}
	}
}

func multiplex(logger chassis.Logger, conn *net.UDPConn, stunAddr net.Addr, stunConn io.Reader) {
	// Sending all data from stun client to stun server.
	buf := make([]byte, 1024)
	for {
		n, err := stunConn.Read(buf)
		if err != nil {
			logger.WithError(err).Error("failed to read")
			return
		}
		if _, err = conn.WriteTo(buf[:n], stunAddr); err != nil {
			logger.WithError(err).Error("failed to write")
			return
		}
	}
}

func (c *stunClient) bind(logger chassis.Logger, server string) (address stun.XORMappedAddress, err error) {

	// Allocating local UDP socket that will be used both for STUN and
	// our application data.
	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:0")
	if err != nil {
		logger.WithError(err).Error("failed to resolve local UDP socket")
		return address, err
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		logger.WithError(err).Error("failed to listen on socket")
		return address, err
	}
	c.conn = conn

	// Resolving STUN server address.
	stunAddr, err := net.ResolveUDPAddr("udp4", server)
	if err != nil {
		logger.WithError(err).Error("failed to resolve STUN server address")
		return address, err
	}

	stunL, stunR := net.Pipe()

	client, err := stun.NewClient(stunR)
	if err != nil {
		logger.WithError(err).Error("failed to create STUN client")
		return address, err
	}

	// Starting multiplexing (writing back STUN messages) with de-multiplexing
	// (passing STUN messages to STUN client and processing application
	// data separately).
	//
	// stunL and stunR are virtual connections, see net.Pipe for reference.
	messages := make(chan message)

	go demultiplex(logger, conn, stunL, messages)
	go multiplex(logger, conn, stunAddr, stunL)

	// Getting our "real" IP address from STUN Server.
	// This will create a NAT binding on your provider/router NAT Server,
	// and the STUN server will return allocated public IP for that binding.
	//
	// This can fail if your NAT Server is strict and will use separate ports
	// for application data and STUN
	if err = client.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			logger.WithError(res.Error).Error("failed STUN transaction")
			return
		}
		var xorAddr stun.XORMappedAddress
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			logger.WithError(getErr).Error("failed to get XOR-MAPPED-ADDRESS")
			return
		}
		copyAddr(&address, xorAddr)
	}); err != nil {
		logger.WithError(err).Error("failed STUN transaction")
		return address, err
	}

	// Keep-alive is needed to keep our NAT port allocated.
	// Any ping-pong will work, but we are just making binding requests.
	// Note that STUN Server is not mandatory for keep alive, application
	// data will keep alive that binding too.
	go keepAlive(logger, client)

	// TODO: forward messages to wireguard
	go func() {
		for m := range messages {
			if _, err = conn.WriteTo([]byte(m.text), m.addr); err != nil {
				logger.WithError(err).Error("failed to write")
			}
		}
	}()

	logger.WithField("address", address.String()).Debug("found STUN address")

	return address, nil
}
