# Tunnel

The tunnel service monitors [Wireguard CRD resources](../operator/api/v1/wireguard_types.go) and configures the Linux kernel implementation of Wireguard to match the desired state.

It also connects to remote remote [Locator services](../locator/README.md) to handle peer-to-server discovery via STUN.

More info [here](https://home-cloud.io/blog/on-the-go-architecture/).
