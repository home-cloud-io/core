package host

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/netbirdio/netbird/sharedsock"
	"github.com/pion/stun/v2"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	STUNController interface {
		// Bind creates a persistent connection to the given STUN server from the given port. All data received on this port will
		// be multiplexed between the STUN client and the upstream service listening on the given host port (e.g. Wireguard).
		Bind(port int, server string) error
		// Address returns the current STUN address for the given port (if there is one).
		Address(port int) (stun.XORMappedAddress, error)
		// Connect initializes a short period of connection attempts to the given STUN address of a peer from the
		// given port. This opens a hole in the NAT for inbound connection attempts from the peer.
		Connect(port int, address net.Addr)
		// Cancel destructs an existing STUN binding on the given port.
		Cancel(port int) error
	}
	stunController struct {
		logger   chassis.Logger
		bindings map[int]*stunBinding
	}
	stunBinding struct {
		cancel  context.CancelFunc
		port    int
		server  string
		client  *stun.Client
		conn    net.PacketConn
		address stun.XORMappedAddress
	}
)

const (
	keepAliveInterval = 5 * time.Second
	connectDuration   = 10 * time.Second
	connectInterval   = 1 * time.Second
)

func NewSTUNController(logger chassis.Logger) STUNController {
	return &stunController{
		logger: logger,
		bindings: make(map[int]*stunBinding),
	}
}

func (c *stunController) Bind(port int, server string) (err error) {
	c.logger.WithField("port", port).WithField("stun_server", server).Info("binding to STUN server")
	ctx, cancel := context.WithCancel(context.Background())
	binding := &stunBinding{
		cancel: cancel,
		port:   port,
		server: server,
	}

	logger := c.logger.WithFields(chassis.Fields{
		"host_port":   port,
		"stun_server": server,
	})

	rawSocket, err := sharedsock.Listen(port, sharedsock.NewIncomingSTUNFilter())
	if err != nil {
		logger.WithError(err).Error("failed to listen on shared socket")
		return err
	}
	binding.conn = rawSocket

	// resolve the given STUN server address
	stunAddr, err := net.ResolveUDPAddr("udp4", server)
	if err != nil {
		logger.WithError(err).Error("failed to resolve STUN server address")
		return err
	}
	logger.WithField("stun_address", stunAddr).Info("resolved STUN server to address")

	// in-memory network pipe between stun client and multiplexer
	stunL, stunR := net.Pipe()

	// create new STUN client
	client, err := stun.NewClient(stunR)
	if err != nil {
		logger.WithError(err).Error("failed to create STUN client")
		return err
	}
	binding.client = client

	// start de/multiplexing
	go demultiplex(ctx, logger, rawSocket, stunL)
	go multiplex(ctx, logger, rawSocket, stunAddr, stunL)

	// attempt to bind to the STUN server and aquire our STUN address
	err = binding.bind()
	if err != nil {
		logger.WithError(err).Error("failed to bind to STUN server")
		return err
	}

	// keep binding alive until canceled
	go keepAlive(ctx, logger, binding)

	// save binding config
	c.bindings[port] = binding

	logger.WithField("address", binding.address.String()).Info("finished binding to STUN server")

	return nil
}

func (c *stunController) Address(port int) (address stun.XORMappedAddress, err error) {
	binding, ok := c.bindings[port]
	if !ok {
		return address, errors.New("no STUN binding for given port")
	}
	return binding.address, nil
}

func (c *stunController) Connect(port int, address net.Addr) {
	binding, ok := c.bindings[port]
	if !ok {
		c.logger.WithField("host_port", port).WithField("remote_address", address.String()).Error("no STUN binding for given port")
		return
	}

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
			_, err := binding.conn.WriteTo([]byte(msg), address)
			if err != nil {
				log.Warn("failed to send connect message to peer")
			}
		}
	}
}

func (c *stunController) Cancel(port int) error {
	binding, ok := c.bindings[port]
	if !ok {
		c.logger.WithField("host_port", port).Warn("no STUN binding for given port")
		return nil
	}

	// cancel binding context to kill child routines
	binding.cancel()

	// close binding stun client
	err := binding.client.Close()
	if err != nil {
		return err
	}

	// remove binding
	delete(c.bindings, port)

	return nil
}

// keepAlive sends periodic binding requests to the STUN server to maintain the opening in the NAT
func keepAlive(ctx context.Context, logger chassis.Logger, binding *stunBinding) {
	ticker := time.NewTicker(keepAliveInterval)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("stopping STUN keep alive")
			return
		case <-ticker.C:
			err := binding.bind()
			if err != nil {
				logger.WithError(err).Error("failed STUN transaction")
				return
			}
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
			logger.Debug("stopping STUN demultiplexer")
			return
		default:
			size, addr, err := conn.ReadFrom(buf)
			if err != nil {
				logger.Errorf("error while reading packet from the shared socket: %s", err)
				continue
			}
			logger.WithField("packet_size", size).WithField("address", addr).Debug("read a STUN packet")
			if _, err = stunConn.Write(buf[:size]); err != nil {
				logger.WithError(err).Error("failed to write")
				return
			}
		}
	}
}

// multiplex reads messages from the given STUN connection and writes them to the given STUN address (server) using the
// provided UDP connection.
func multiplex(ctx context.Context, logger chassis.Logger, conn net.PacketConn, stunAddr net.Addr, stunConn io.Reader) {
	// Sending all data from stun client to stun server.
	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("stopping STUN multiplexer")
			return
		default:
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
}

// bind wraps the Do() method on the STUN client within the binding so that the address is updated on each binding request.
func (b *stunBinding) bind() error {
	var eventErr error
	err := b.client.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			eventErr = res.Error
		}

		// parse the returned address from the response
		var foundAddress stun.XORMappedAddress
		err := foundAddress.GetFrom(res.Message)
		if err != nil {
			eventErr = err
		}

		// save the found address
		copyAddress(&b.address, foundAddress)
	})
	if err != nil {
		return err
	}
	return eventErr
}

func copyAddress(dst *stun.XORMappedAddress, src stun.XORMappedAddress) {
	dst.IP = append(dst.IP, src.IP...)
	dst.Port = src.Port
}
