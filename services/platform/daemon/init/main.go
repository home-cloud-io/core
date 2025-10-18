package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

var (
	dirs = []string{
		"./tmp/mnt/k8s-pvs/",
	}
	files = map[string]string{
		"auto-install/home-cloud/daemon/config.yaml":     "./tmp/etc/home-cloud/config.yaml",
		"auto-install/home-cloud/daemon/migrations.yaml": "./tmp/etc/home-cloud/migrations.yaml",
		"auto-install/home-cloud/daemon/default.nix":     "./tmp/etc/nixos/home-cloud/daemon/default.nix",
		"auto-install/configuration.nix":                 "./tmp/etc/nixos/configuration.nix",
		"auto-install/hardware/generic.nix":              "./tmp/etc/nixos/hardware-configuration.nix",
		"auto-install/vars.nix":                          "./tmp/etc/nixos/vars.nix",
		"auto-install/home-cloud/draft.yaml":             "./tmp/var/lib/rancher/k3s/server/manifests/draft.yaml",
		"auto-install/home-cloud/gateway-api.yaml":       "./tmp/var/lib/rancher/k3s/server/manifests/gateway-api.yaml",
		"auto-install/home-cloud/istio.yaml":             "./tmp/var/lib/rancher/k3s/server/manifests/istio.yaml",
		"auto-install/home-cloud/operator.yaml":          "./tmp/var/lib/rancher/k3s/server/manifests/operator.yaml",
		"auto-install/home-cloud/server.yaml":            "./tmp/var/lib/rancher/k3s/server/manifests/server.yaml",
	}
)

func main() {
	client := &http.Client{}

	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Dir(dir), 0755)
		if err != nil {
			panic(err)
		}
	}

	for src, dest := range files {
		downloadFile(client, src, dest)
	}
}

func downloadFile(client *http.Client, src, dest string) {
	err := os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/home-cloud-io/os/contents/%s?ref=main", src), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Accept", "application/vnd.github.VERSION.raw")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(dest, data, 0666)
	if err != nil {
		panic(err)
	}
}
