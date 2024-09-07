package host

import (
	"context"
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
	ctx := context.Background()
	config := chassis.GetConfig()
	domain := config.GetString("daemon.domain")
	if domain == "" {
		p.logger.Panic("domain not set")
	}
	address := config.GetString("domain.address")
	hostnames := config.GetStringSlice("daemon.hosts")
	for _, hostname := range hostnames {
		go func(){
			cmd := exec.Command("avahi-publish", "-a", "-R", hostname, address)
			err := execute.ExecuteCommand(ctx, cmd)
			if err != nil {
				p.logger.WithField("hostname", hostname).Panic("failed to publish mDNS")
			}
		}()
	}
}