package host

import (
	"net"
	"os"
	"strings"

	"github.com/spf13/viper"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Runner interface {
		OnStart()
	}
	runner struct {
		logger chassis.Logger
	}
)

const (
	ipAddressConfigKey = "daemon.ip_address"
)

var (
	ipFiles = []string{
		"/var/lib/rancher/k3s/server/manifests/draft.yaml",
	}
)

func NewRunner(logger chassis.Logger) Runner {
	return runner{
		logger: logger,
	}
}

func (r runner) OnStart() {
	config := chassis.GetConfig()

	currentIP, err := getOutboundIP()
	if err != nil {
		r.logger.WithError(err).Error("failed to get outbound ip")
		return
	}

	previousIP := config.GetString(ipAddressConfigKey)
	if previousIP == currentIP {
		r.logger.WithField("ip", currentIP).Info("ip unchanged")
		return
	}

	r.logger.WithField("ip", currentIP).Info("setting new ip")
	err = setIP(previousIP, currentIP)
	if err != nil {
		r.logger.WithError(err).Error("failed to set ip")
		return
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

func setIP(old, new string) error {
	// write ip to daemon config
	viper.Set(ipAddressConfigKey, new)
	err := viper.WriteConfig()
	if err != nil {
		return err
	}

	for _, fileName := range ipFiles {
		// read file
		input, err := os.ReadFile(fileName)
		if err != nil {
			return err
		}

		// replace text line by line
		lines := strings.Split(string(input), "\n")
		for i, line := range lines {
			lines[i] = strings.ReplaceAll(line, old, new)
		}

		// write file
		output := strings.Join(lines, "\n")
		err = os.WriteFile(fileName, []byte(output), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}