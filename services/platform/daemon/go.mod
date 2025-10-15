module github.com/home-cloud-io/core/services/platform/daemon

go 1.24.0

toolchain go1.24.4

replace github.com/home-cloud-io/core/api => ../../../api

// replace github.com/steady-bytes/draft/pkg/chassis => ../../../../../steady-bytes/draft/pkg/chassis

replace golang.zx2c4.com/wireguard => github.com/netbirdio/wireguard-go v0.0.0-20241230120307-6a676aebaaf6

require (
	connectrpc.com/connect v1.18.1
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/home-cloud-io/core/api v0.8.7
	github.com/mackerelio/go-osstat v0.2.5
	github.com/netbirdio/netbird v0.39.2
	github.com/pion/stun/v2 v2.0.0
	github.com/steady-bytes/draft/api v1.1.0
	github.com/steady-bytes/draft/pkg/chassis v0.4.5
	github.com/steady-bytes/draft/pkg/loggers v0.2.3
	golang.org/x/crypto v0.37.0
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20241231184526-a9ab2273dd10
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.36.7
	gopkg.in/yaml.v3 v3.0.1
)

require (
	connectrpc.com/grpcreflect v1.2.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/cloudevents/sdk-go/binding/format/protobuf/v2 v2.15.0 // indirect
	github.com/envoyproxy/go-control-plane v0.12.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/fatih/color v1.14.1 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack/v2 v2.1.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/raft v1.6.0 // indirect
	github.com/hashicorp/raft-boltdb/v2 v2.3.0 // indirect
	github.com/libp2p/go-netroute v0.2.1 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/pion/dtls/v2 v2.2.10 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/transport/v2 v2.2.4 // indirect
	github.com/pion/transport/v3 v3.0.1 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/rs/cors v1.10.1 // indirect
	github.com/rs/zerolog v1.32.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/spf13/viper v1.18.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/things-go/go-socks5 v0.0.4 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	golang.org/x/time v0.9.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	golang.zx2c4.com/wireguard v0.0.0-20231211153847-12269c276173 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gvisor.dev/gvisor v0.0.0-20231020174304-b8a429915ff1 // indirect
)
