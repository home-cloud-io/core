module github.com/home-cloud-io/core/services/platform/daemon

go 1.25.3

replace github.com/home-cloud-io/core/api => ../../../api

// replace github.com/steady-bytes/draft/pkg/chassis => ../../../../../steady-bytes/draft/pkg/chassis

replace golang.zx2c4.com/wireguard => github.com/netbirdio/wireguard-go v0.0.0-20241230120307-6a676aebaaf6

require (
	connectrpc.com/connect v1.18.1
	github.com/home-cloud-io/core/api v0.8.7
	github.com/mackerelio/go-osstat v0.2.5
	github.com/steady-bytes/draft/pkg/chassis v0.4.5
	github.com/steady-bytes/draft/pkg/loggers v0.2.3
	google.golang.org/protobuf v1.36.7
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
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack/v2 v2.1.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/raft v1.6.0 // indirect
	github.com/hashicorp/raft-boltdb/v2 v2.3.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/rs/cors v1.10.1 // indirect
	github.com/rs/zerolog v1.32.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/spf13/viper v1.18.2 // indirect
	github.com/steady-bytes/draft/api v1.1.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
