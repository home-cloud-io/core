package host

import (
	"context"
	"io"
	"net"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"

	"github.com/google/uuid"
	"github.com/netbirdio/netbird/sharedsock"
	"github.com/pion/stun/v2"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	STUNClient interface {
		// Bind restarts the STUN client using the given STUN server
		Bind(server string) (stun.XORMappedAddress, error)
		// Connect initializes a short period of connection attempts to the given STUN address of a peer.
		// This opens a hole in the NAT for inbound connection attempts from the peer.
		Connect(address net.Addr)
	}
	stunClient struct {
		logger chassis.Logger
		client *stun.Client
		conn   net.PacketConn
	}
)

type message struct {
	body []byte
	addr net.Addr
}

const (
	keepAliveInterval = 5 * time.Second
	connectDuration   = 10 * time.Second
	connectInterval   = 1 * time.Second
)

func NewSTUNClient(logger chassis.Logger) STUNClient {
	return &stunClient{
		logger: logger,
	}
}

func (c *stunClient) Bind(server string) (address stun.XORMappedAddress, err error) {
	// get current settings
	settings := &v1.LocatorSettings{}
	err = chassis.GetConfig().UnmarshalKey(LocatorSettingsKey, settings)
	if err != nil {
		return address, err
	}
	// update settings
	settings.StunServerAddress = server
	err = chassis.GetConfig().SetAndWrite(LocatorSettingsKey, settings)
	if err != nil {
		return address, err
	}

	return c.bind(c.logger, server)
}

func (c *stunClient) Connect(address net.Addr) {
	deadline := time.After(connectDuration)
	for {
		select {
		case <-deadline:
			c.logger.Debug("finished attempt to open connection to peer")
			return
		case <-time.After(connectInterval):
			msg := uuid.New().String()
			log := c.logger.WithFields(chassis.Fields{
				"msg":     msg,
				"address": address,
			})
			log.Debug("sending message")
			_, err := c.conn.WriteTo([]byte(msg), address)
			if err != nil {
				log.Warn("failed to send connect message to peer")
			}
		}
	}
}

// keepAlive sends periodic binding requests to the STUN server to maintain the opening in the NAT
func keepAlive(logger chassis.Logger, c *stun.Client) {
	t := time.NewTicker(keepAliveInterval)
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

// demultiplex reads messages from given UDP connection, checks if the messages are STUN messages and writes them to the given STUN writer if so. Otherwise,
// the messages are treated as application data and are sent to the given message channel.
func demultiplex(ctx context.Context, logger chassis.Logger, conn net.PacketConn, stunConn io.Writer) {
	buf := make([]byte, 1500)
	for {
		select {
		case <-ctx.Done():
			logger.Debugf("stopped reading from the shared socket")
			return
		default:
			size, addr, err := conn.ReadFrom(buf)
			if err != nil {
				logger.Errorf("error while reading packet from the shared socket: %s", err)
				continue
			}
			logger.Infof("read a STUN packet of size %d from %s", size, addr.String())
			if _, err = stunConn.Write(buf[:size]); err != nil {
				logger.WithError(err).Error("failed to write")
				return
			}
		}
	}
}

// multiplex reads messages from the given STUN connection and writes them to the given STUN address (server) using the
// provided UDP connection.
func multiplex(logger chassis.Logger, conn net.PacketConn, stunAddr net.Addr, stunConn io.Reader) {
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

// bind establishes a persistent connection with the given STUN server, initializes multiplexing for application data and returns
// the found STUN address.
func (c *stunClient) bind(logger chassis.Logger, server string) (address stun.XORMappedAddress, err error) {
	// TODO: pass this in?
	ctx := context.Background()

	port := 51820
	rawSock, err := sharedsock.Listen(port, sharedsock.NewIncomingSTUNFilter())
	if err != nil {
		panic(err)
	}
	c.conn = rawSock

	// resolve the given STUN server address
	stunAddr, err := net.ResolveUDPAddr("udp4", server)
	if err != nil {
		logger.WithError(err).Error("failed to resolve STUN server address")
		return address, err
	}

	stunL, stunR := net.Pipe()

	// attempt to close existing client before creating new one
	if c.client != nil {
		err := c.client.Close()
		if err != nil {
			c.logger.WithError(err).Error("failed to close client")
			return address, err
		}
	}

	// create new STUN client
	client, err := stun.NewClient(stunR)
	if err != nil {
		logger.WithError(err).Error("failed to create STUN client")
		return address, err
	}

	// start de/multiplexing
	go demultiplex(ctx, logger, rawSock, stunL)
	go multiplex(logger, rawSock, stunAddr, stunL)

	// attempt to bind to the STUN server and aquire our STUN address
	err = client.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		// check for error during bind
		if res.Error != nil {
			logger.WithError(res.Error).Error("failed STUN transaction")
			return
		}

		// parse the returned address from the response
		var foundAddress stun.XORMappedAddress
		err := foundAddress.GetFrom(res.Message)
		if err != nil {
			logger.WithError(err).Error("failed to get XOR-MAPPED-ADDRESS")
			return
		}

		// save the found address for future use
		copyAddress(&address, foundAddress)
	})
	if err != nil {
		logger.WithError(err).Error("failed to bind to STUN server")
		return address, err
	}

	// TODO: pass in context so we can cancel this
	go keepAlive(logger, client)

	logger.WithField("address", address.String()).Info("finished binding to STUN server")

	return address, nil
}

func copyAddress(dst *stun.XORMappedAddress, src stun.XORMappedAddress) {
	dst.IP = append(dst.IP, src.IP...)
	dst.Port = src.Port
}
