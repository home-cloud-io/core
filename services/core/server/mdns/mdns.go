package mdns

import (
	"net"

	"github.com/pion/mdns/v2"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/net/ipv4"
)

func ServeMDNS(logger chassis.Logger) {
	config := chassis.GetConfig()

	addr4, err := net.ResolveUDPAddr("udp4", mdns.DefaultAddressIPv4)
	if err != nil {
		logger.WithError(err).Fatal("failed to resolve UDP address")
	}

	l4, err := net.ListenUDP("udp4", addr4)
	if err != nil {
		logger.WithError(err).Fatal("failed to listen on UDP")
	}

	_, err = mdns.Server(ipv4.NewPacketConn(l4), nil, &mdns.Config{
		LocalNames:   []string{"home-cloud.local"},
		LocalAddress: net.ParseIP(config.GetString("HOST_IP")),
	})
	if err != nil {
		logger.WithError(err).Fatal("failed to serve mDNS")
	}
}
