package mdns

import (
	"context"
	"net"

	"github.com/pion/mdns/v2"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/net/ipv4"
)

type (
	Server interface {
		Serve(ctx context.Context) error
		Close(ctx context.Context) error
		AddHost(ctx context.Context, host string) error
		RemoveHost(ctx context.Context, host string) error
	}
	server struct {
		logger chassis.Logger
		conn   *mdns.Conn
		hosts  []string
	}
)

func New(logger chassis.Logger) Server {
	return &server{
		logger: logger,
		hosts:  make([]string, 0),
	}
}

func (s *server) Serve(ctx context.Context) error {

	addr4, err := net.ResolveUDPAddr("udp4", mdns.DefaultAddressIPv4)
	if err != nil {
		return err
	}

	l4, err := net.ListenUDP("udp4", addr4)
	if err != nil {
		return err
	}

	if len(s.hosts) == 0 {
		return s.Close(ctx)
	}

	// server hosts
	conn, err := mdns.Server(ipv4.NewPacketConn(l4), nil, &mdns.Config{
		LocalNames:   s.hosts,
		LocalAddress: net.ParseIP("192.168.1.184"),
		// LocalAddress: net.ParseIP(os.Getenv("HOST_IP")),
	})
	if err != nil {
		return err
	}

	// close old conn
	if s.conn != nil {
		err := s.Close(ctx)
		if err != nil {
			s.logger.Infof("failed to close connection: %v\n", err)
		}
	}

	// assign new conn
	s.conn = conn

	return nil
}

func (s *server) Close(ctx context.Context) error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

func (s *server) AddHost(ctx context.Context, host string) error {
	for _, h := range s.hosts {
		// if host is already registered just ignore
		if h == host {
			return nil
		}
	}
	s.hosts = append(s.hosts, host)
	return s.Serve(ctx)
}

func (s *server) RemoveHost(ctx context.Context, host string) error {
	hosts := []string{}
	for _, h := range s.hosts {
		if h != host {
			hosts = append(hosts, h)
		}
	}
	return s.Serve(ctx)
}
