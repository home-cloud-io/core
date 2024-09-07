package host

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/home-cloud-io/core/services/platform/daemon/execute"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	DNSPublisher interface {
		AddHost(ctx context.Context, hostname string)
		RemoveHost(hostname string) error
		Start()
	}
	dnsPublisher struct {
		logger  chassis.Logger
		domain  string
		address string
		// a map of FQDNs to their respective cancel functions
		cancels map[string]context.CancelFunc
	}
)

func NewDNSPublisher(logger chassis.Logger) DNSPublisher {
	return &dnsPublisher{
		logger:  logger,
		cancels: map[string]context.CancelFunc{},
	}
}

func (p *dnsPublisher) Start() {
	ctx := context.Background()
	p.logger.Info("starting mDNS publishing")

	// set domain
	config := chassis.GetConfig()
	p.domain = config.GetString("daemon.domain")
	if p.domain == "" {
		p.logger.Panic("domain not set")
	}

	// set address
	address, err := getOutboundIP()
	if err != nil {
		p.logger.WithError(err).Panic("failed to get outbound IP address")
	}
	p.logger.WithField("address", address).Info("found outbound IP address")
	p.address = address

	// start initial set of hosts from config
	hostnames := config.GetStringSlice("daemon.hostnames")
	for _, hostname := range hostnames {
		p.AddHost(ctx, hostname)
	}
}

func (p *dnsPublisher) AddHost(ctx context.Context, hostname string) {
	c, cancel := context.WithCancel(ctx)
	fqdn := p.buildFQDN(hostname)
	logger := p.logger.WithField("fqdn", fqdn)

	logger.Info("adding host to mDNS")

	p.cancels[fqdn] = cancel
	go publish(c, logger, fqdn, p.address)

	logger.Info("host added to mDNS")
}

func (p *dnsPublisher) RemoveHost(hostname string) error {
	fqdn := p.buildFQDN(hostname)
	logger := p.logger.WithField("fqdn", fqdn)

	logger.Info("removing host from mDNS")

	// find and canel context associated with the given hostname
	f, ok := p.cancels[fqdn]
	if !ok {
		return fmt.Errorf("host not found to remove")
	}
	f()

	logger.Info("host removed from mDNS")
	return nil
}

func (p *dnsPublisher) buildFQDN(hostname string) string {
	return fmt.Sprintf("%s.%s", hostname, p.domain)
}

func publish(ctx context.Context, logger chassis.Logger, fqdn, address string) {
	for {
		// if the context is cancelled just return
		if ctx.Err() != nil {
			return
		}
		logger = logger.WithField("address", address)
		cmd := exec.Command("avahi-publish", "-a", "-R", fqdn, address)
		err := execute.ExecuteCommand(ctx, cmd)
		if err != nil {
			logger.WithError(err).Error("failed to publish mDNS")
			// TODO: notify server of failure
		}
		// wait a few seconds before trying again
		time.Sleep(5 * time.Second)
	}
}

// Get preferred outbound ip of this machine
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "home-cloud.io:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
