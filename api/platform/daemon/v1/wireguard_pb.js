// @generated by protoc-gen-es v1.10.0 with parameter "target=js"
// @generated from file platform/daemon/v1/wireguard.proto (package platform.daemon.v1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import { proto3 } from "@bufbuild/protobuf";

/**
 * @generated from message platform.daemon.v1.WireguardConfig
 */
export const WireguardConfig = /*@__PURE__*/ proto3.makeMessageType(
  "platform.daemon.v1.WireguardConfig",
  () => [
    { no: 1, name: "interfaces", kind: "message", T: WireguardInterface, repeated: true },
  ],
);

/**
 * @generated from message platform.daemon.v1.WireguardInterface
 */
export const WireguardInterface = /*@__PURE__*/ proto3.makeMessageType(
  "platform.daemon.v1.WireguardInterface",
  () => [
    { no: 1, name: "id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "name", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "private_key", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 4, name: "ips", kind: "scalar", T: 9 /* ScalarType.STRING */, repeated: true },
    { no: 5, name: "listen_port", kind: "scalar", T: 13 /* ScalarType.UINT32 */ },
    { no: 6, name: "peers", kind: "message", T: WireguardPeer, repeated: true },
  ],
);

/**
 * @generated from message platform.daemon.v1.WireguardPeer
 */
export const WireguardPeer = /*@__PURE__*/ proto3.makeMessageType(
  "platform.daemon.v1.WireguardPeer",
  () => [
    { no: 1, name: "public_key", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "allowed_ips", kind: "scalar", T: 9 /* ScalarType.STRING */, repeated: true },
  ],
);
