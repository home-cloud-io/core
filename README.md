# Home Cloud Core

This repository contains the core components that make up the Home Cloud platform. For example

- **Server**: the primary service that manages users, settings, and hosts the web interface
- **Operator**: a Kubernetes operator responsible for managing user application lifecycle
- **Daemon**: a system service that manages NixOS configuration and low-level host commands (like reboots)
- **Locator**: a zero-trust service discovery engine to enable remote access to Home Cloud servers when not at home

## Getting Started

To work on the Home Cloud core platform you'll need a couple of things installed:

* [Go](https://golang.org/doc/install) v1.23+
* [Docker](https://docs.docker.com/get-docker/)
* [Node via NVM](https://github.com/nvm-sh/nvm?tab=readme-ov-file#installing-and-updating)

This repository is built on top of the [Draft framework](https://github.com/steady-bytes/draft) for distributed systems. You don't need to be an expert with Draft to work with the Home Cloud core platform, but you'll need at least the `dctl` CLI tool. Let's install it now:

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

### draft services

You can start all required services locally in Docker with:

```shell
dctl infra init # only need to run this when updating docker images
dctl infra start
```

### local k3s cluster

You'll need a k3s cluster for development. You can create a local k3s cluster using [k3d (k3s in docker)](https://k3d.io/stable/). Follow the installation directions on their site to get the `k3d` CLI installed. Now you can start a basic k3d cluster with:

```sh
k3d cluster create --api-port 6550 -p '9080:80@loadbalancer' -p '9443:443@loadbalancer' --agents 1 --k3s-arg '--disable=traefik@server:*' home-cloud
```

You can create the `home-cloud-system` namespace which will be needed later:

```sh
kubectl create namespace home-cloud-system
```

### istio

Istio runs as a service mesh between all Home Cloud resources in Kubernetes. First install the k8s Gateway API:

```sh
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml
```

We need to install Istio in Ambient mode. You can do this using the [official documentation](https://istio.io/latest/docs/ambient/install/) for any platform, but if you're using the k3d cluster we created above, you can just run the below command for an automated install:

```sh
kubectl apply -f development/istio.yaml
```

### server

Before running the server, we need to first build the web client that is hosted by the server. Navigate to `services/platform/server/web-client` and build the web client:

```sh
cd services/platform/server/web-client
npm install
npm run build
```

To run the server locally, first navigate up a directory (`cd ..`), then you can start the server (you may need to change the KUBECONFIG path):

```sh
KUBECONFIG=~/.kube/config go run main.go
```

The server will now connect to `blueprint` running in Docker (we ran it with `dctl infra start`) and connect to your local k3d cluster using your local kubeconfig.

### daemon

To run the daemon locally, first navigate to the `services/platform/daemon` directory (in a new terminal if you're running the server from before) and initialize the local filesystem:

```sh
go run init/main.go
```

Now you can start the daemon with:

```sh
go run main.go
```

### operator

First install the operator's CRDs to the cluster:

```sh
cd services/platform/operator
kubectl apply -f config/crd/bases/home-cloud.io_apps.yaml
```

Now you can run the operator:

```sh
DRAFT_SERVICE_ENV=test go run main.go
```

### web client

If you're developing the web client, you can run this in development mode:

```sh
cd services/platform/server/web-client
npm start
```

This will open your browser to the web client running locally and proxying all requests to the home cloud server running on `localhost:8000`.
