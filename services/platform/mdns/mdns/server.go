package mdns

import (
	"context"
	"fmt"
	"net"
	"slices"

	"github.com/pion/mdns/v2"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
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

const (
	HostIPConfigKey     = "mdns.host_ip"
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
	s.logger.WithField("address", addr4).Info("serving mDNS on IPv4")

	l4, err := net.ListenUDP("udp4", addr4)
	if err != nil {
		return err
	}

	addr6, err := net.ResolveUDPAddr("udp6", mdns.DefaultAddressIPv6)
	if err != nil {
		return err
	}
	s.logger.WithField("address", addr6).Info("serving mDNS on IPv6")

	l6, err := net.ListenUDP("udp6", addr6)
	if err != nil {
		return err
	}

	if len(s.hosts) == 0 {
		return s.Close(ctx)
	}

	// server hosts
	hostIPString := chassis.GetConfig().GetString(HostIPConfigKey)
	hostIP := net.ParseIP(hostIPString)
	if hostIP == nil {
		return fmt.Errorf("invalid host IP: %s", hostIPString)
	}
	conn, err := mdns.Server(ipv4.NewPacketConn(l4), ipv6.NewPacketConn(l6), &mdns.Config{
		LocalNames:   s.hosts,
		LocalAddress: hostIP,
	})
	if err != nil {
		return err
	}
	s.logger.WithFields(chassis.Fields{
		"address": hostIP,
		"hosts":   s.hosts,
	}).Info("registered host IP")

	// close old conn
	if s.conn != nil {
		err := s.Close(ctx)
		if err != nil {
			s.logger.WithError(err).Error("failed to close connection")
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
	if slices.Contains(s.hosts, host) {
		return nil
	}
	s.logger.WithField("host", host).Info("adding host")
	s.hosts = append(s.hosts, host)
	return s.Serve(ctx)
}

func (s *server) RemoveHost(ctx context.Context, host string) error {
	s.logger.WithField("host", host).Info("removing host")
	// TODO: should change this to a thread-safe map using a mutex?
	hosts := []string{}
	for _, h := range s.hosts {
		if h != host {
			hosts = append(hosts, h)
		}
	}
	s.hosts = hosts
	return s.Serve(ctx)
}
