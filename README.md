# Home Cloud Core

The easy-to-use solution that enables you to say goodbye to the high-cost, privacy nightmare of Big Tech services so that you can finally take back control over your digital life!

For more info: https://home-cloud.io/

## Contents

This repository contains the core components that make up the Home Cloud platform. These include:

- [**daemon**](./services/platform/daemon/README.md): a system service that manages a [Talos](https://talos.dev) installation and low-level host commands (like reboots).
- [**locator**](./services/platform/locator/README.md): a zero-trust service discovery engine to enable remote access to Home Cloud servers when not at home.
- [**mdns**](./services/platform/mdns/README.md): a lightweight [mDNS](https://en.wikipedia.org/wiki/Multicast_DNS) server which creates mDNS entries based off of Kubernetes Service annotations.
- [**operator**](./services/platform/operator/README.md): a Kubernetes operator which manages the Home Cloud installation itself as well as user installed Apps. It also hosts the web client and API that manages users, settings, and all other user actions.
- [**tunnel**](./services/platform/tunnel/README.md): a small Kubernetes operator which uses the [**locator**](./services/platform/locator/README.md) to create Wireguard tunnels to mobile devices.

## Requirements

To work on the Home Cloud core platform you'll need a couple of things installed:

* [Go](https://golang.org/doc/install) - v1.25+
* [Docker](https://docs.docker.com/get-docker/)
* [Node (recommend nvm)](https://github.com/nvm-sh/nvm)
* [talosctl](https://docs.siderolabs.com/talos/latest/getting-started/quickstart)
* [kubectl](https://kubernetes.io/docs/tasks/tools/)

## Getting Started

This repository is built on top of the [Draft framework](https://github.com/steady-bytes/draft) for distributed systems. You don't need to be an expert with Draft to work with the Home Cloud core platform, but you'll need at least the `dctl` CLI tool.

Let's install it now:

```shell
go install github.com/steady-bytes/draft/tools/dctl@latest
```

We'll need to import this project as a context into `dctl` so it can manage things for us. After cloning the repo run the below command from the root of the repo:

```shell
dctl context import
```

Let's do a quick test of building the Home Cloud API protobufs:

```shell
dctl api init
dctl api build
```

## Development

### local Talos cluster

You'll need a Talos cluster for development. We'll create one to run in Docker locally:

```sh
talosctl cluster create docker --workers 0
```

<!-- TODO: enable workloads on control-plane node -->

Now create the `home-cloud-system` namespace which will be needed later:

```sh
kubectl create namespace home-cloud-system
```

### CRDs

First install the Home Cloud CRDs to the cluster:

```sh
cd services/platform/operator
kubectl apply -f services/platform/operator/config/crd/bases/home-cloud.io_apps.yaml
kubectl apply -f services/platform/operator/config/crd/bases/home-cloud.io_installs.yaml
kubectl apply -f services/platform/operator/config/crd/bases/home-cloud.io_wireguards.yaml
```

### server

Before running the server, we need to first build the web client that is hosted by the server:

```sh
cd services/platform/server/web-client
npm install
npm run build
```

Now you can start the server (you may need to change the KUBECONFIG path):

```sh
cd ..
KUBECONFIG=~/.kube/config go run main.go
```

### daemon

You can run the daemon with:

```sh
cd services/platform/daemon
go run main.go
```

### operator

You can run the operator with:

```sh
cd services/platform/operator
go run main.go
```

### web client

If you're developing the web client, you can run it in development mode:

```sh
cd services/platform/server/web-client
npm start
```

This will open your browser to the web client running locally on `localhost:3000` and proxying all requests to the home cloud server running on `localhost:8000`.

### Install Steps

```bash
# install talosctl
curl -sL https://talos.dev/install | sh
# install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

# get from machine display out
export MACHINE_IP="192.168.1.183"

# generate config files
talosctl gen secrets -o secrets.yaml
talosctl gen config --with-secrets secrets.yaml home-cloud https://${MACHINE_IP}:6443

# update `install.disk` in controlplane.yaml
talosctl --nodes ${MACHINE_IP} get disks --insecure

# add home cloud specific config to controlplane.yaml

# apply config
talosctl apply-config --insecure --nodes ${MACHINE_IP} --file controlplane.yaml

# save config
talosctl config merge ./talosconfig

# set endpoint
talosctl config endpoint ${MACHINE_IP}
talosctl --nodes ${MACHINE_IP} get disks --insecure

# install kubernetes
talosctl bootstrap --nodes ${MACHINE_IP}

# save kubeconfig
talosctl kubeconfig kubeconfig --nodes ${MACHINE_IP}
export KUBECONFIG=./kubeconfig

# check connection
kubectl get nodes

# install home cloud manifests
kubectl apply -f https://github.com/home-cloud-io/core/releases/latest/download/crds.yaml
kubectl apply -f https://github.com/home-cloud-io/core/releases/latest/download/operator.yaml
kubectl apply -f https://github.com/home-cloud-io/core/releases/latest/download/install.yaml

# wait until the operator is ready
kubectl get deployment operator -n home-cloud-system -w
```
