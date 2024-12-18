package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	sv1 "github.com/home-cloud-io/core/api/platform/server/v1"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"
)

func main() {
	ctx := context.Background()

	kvclient.Init()

	settings := &sv1.DeviceSettings{}
	err := kvclient.Get(ctx, kvclient.DEFAULT_DEVICE_SETTINGS_KEY, settings)
	if err != nil {
		log.Panic(err)
	}

	fmt.Println(settings)

	// get the server's wireguard wgConfig
	wgConfig := &dv1.WireguardConfig{}
	err = kvclient.Get(ctx, kvclient.WIREGUARD_CONFIG_KEY, wgConfig)
	if err != nil {
		log.Panic(err)
	}

	fmt.Println(wgConfig)

	fmt.Println("Enter the peer's public key:")
	var i string
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		i = scanner.Text()
	}

	if i == "" {
		log.Panicf("No peer given")
	}

	wgConfig.Interfaces[0].Peers = append(wgConfig.Interfaces[0].Peers, &dv1.WireguardPeer{
		PublicKey: i,
	})

	_, err = kvclient.Set(ctx, kvclient.WIREGUARD_CONFIG_KEY, wgConfig)
	if err != nil {
		log.Panic(err)
	}
}
