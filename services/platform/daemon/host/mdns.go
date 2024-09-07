package host

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/home-cloud-io/core/services/platform/daemon/execute"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	DNSPublisher interface {
		// AddHost(hostname string) error
		// RemoveHost(hostname string) error
		Start()
	}
	dnsPublisher struct {
		logger chassis.Logger
	}
)

func NewDNSPublisher(logger chassis.Logger) DNSPublisher {
	return &dnsPublisher{
		logger: logger,
	}
}

func (p *dnsPublisher) Start() {
	p.logger.Info("starting mDNS publishing")
	ctx := context.Background()
	config := chassis.GetConfig()
	domain := config.GetString("daemon.domain")
	if domain == "" {
		p.logger.Panic("domain not set")
	}
	address := config.GetString("daemon.address")
	hostnames := config.GetStringSlice("daemon.hostnames")
	for _, hostname := range hostnames {
		go func() {
			fqdn := fmt.Sprintf("%s.%s", hostname, domain)
			logger := p.logger.WithFields(chassis.Fields{
				"hostname": fqdn,
				"address":  address,
			})
			logger.Info("publishing mDNS")
			cmd := exec.Command("avahi-publish", "-a", "-R", fqdn, address)
			err := execute.ExecuteCommand(ctx, cmd)
			if err != nil {
				p.logger.Panic("failed to publish mDNS")
			}
		}()
	}
}
