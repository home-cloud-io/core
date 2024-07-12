# Home Cloud Core

This repository contains the core components that make up the Home Cloud platform. For example

- **Server**: the primary service that manages users, settings, and hosts the web interface
- **Operator**: a Kubernetes operator responsible for managing user application lifecycle
- **Daemon**: a system service that manages NixOS configuration and low-level host commands (like reboots)

## Getting Started

To work on the Home Cloud core platform you'll need a couple of things installed:

* [Go](https://golang.org/doc/install) v1.21 (we suggest using [gvm](https://github.com/moovweb/gvm) for easier version management)
* [Docker](https://docs.docker.com/get-docker/)

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
