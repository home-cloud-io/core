package main

import (
	"fmt"
	"net"

	"github.com/coreos/go-iptables/iptables"
	"github.com/steady-bytes/draft/pkg/chassis"
	"github.com/steady-bytes/draft/pkg/loggers/zerolog"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"k8s.io/utils/ptr"

	k8sclient "github.com/home-cloud-io/services/platform/wireguard/k8s-client"
)

type (
	Runner struct {
		logger chassis.Logger
	}
)

func main() {
	var (
		logger = zerolog.New()
		runner = &Runner{logger: logger}
	)

	defer chassis.New(logger).
		WithRunner(runner.run).
		Start()
}

func (r *Runner) run() {
	// TODO: get configuration from k8s resources (CRDs and secrets)
	_ = k8sclient.NewClient(r.logger)
	// ctx = context.Background()
	config := chassis.GetConfig()

	wg0 := &netlink.Wireguard{LinkAttrs: netlink.LinkAttrs{Name: "wg0"}}

	err := netlink.LinkAdd(wg0)
	if err != nil {
		panic(err)
	}
	fmt.Println("link added")

	addr, err := netlink.ParseAddr("10.100.0.1/24")
	if err != nil {
		panic(err)
	}
	err = netlink.AddrAdd(wg0, addr)
	if err != nil {
		panic(err)
	}
	fmt.Println("address added")

	wClient, err := wgctrl.New()
	if err != nil {
		panic(err)
	}
	// privateKey, _ := wgtypes.GeneratePrivateKey()
	privateKey, err := wgtypes.ParseKey(config.GetString("wireguard.privateKey"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("private (server): %s\n", privateKey)
	publicKey, err := wgtypes.ParseKey(config.GetString("wireguard.peer.publicKey"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("public (peer): %s\n", publicKey)
	_, peerIPNet, err := net.ParseCIDR("10.100.0.2/24")
	if err != nil {
		panic(err)
	}
	err = wClient.ConfigureDevice("wg0", wgtypes.Config{
		PrivateKey: &privateKey,
		ListenPort: ptr.To(51820),
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey: publicKey,
				AllowedIPs: []net.IPNet{
					*peerIPNet,
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("device configured")

	err = netlink.LinkSetUp(wg0)
	if err != nil {
		panic(err)
	}
	fmt.Println("link set up")

	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		panic(err)
	}
	rule := []string{"-s", "10.100.0.0/24", "-o", "eth0", "-j", "MASQUERADE"}
	exists, err := ipt.Exists("nat", "POSTROUTING", rule...)
	if err != nil {
		panic(err)
	}
	if exists {
		return
	}
	err = ipt.Append("nat", "POSTROUTING", rule...)
	if err != nil {
		panic(err)
	}
	fmt.Println("iptables configured")
}
