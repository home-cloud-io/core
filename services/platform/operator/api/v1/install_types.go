package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallSpec defines the desired state of Install
type InstallSpec struct {
	Version    string         `json:"version"`
	GatewayAPI GatewayAPISpec `json:"gatewayApi,omitempty" yaml:"gatewayApi"`
	Istio      IstioSpec      `json:"istio,omitempty"`
	Server     ServerSpec     `json:"server,omitempty"`
	MDNS       MDNSSpec       `json:"mdns,omitempty"`
	Tunnel     TunnelSpec     `json:"tunnel,omitempty"`
	// TODO: document API
	Daemon   DaemonSpec   `json:"daemon,omitempty"`
	Settings SettingsSpec `json:"settings,omitempty"`
}

type GatewayAPISpec struct {
	// Disabling will not uninstall a previous installation. Since these CRDs are cluster-scoped, this is to avoid
	// breaking an existing installation from another source. You must uninstall manually after disabling.
	Disable bool   `json:"disable,omitempty"`
	URL     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`
}

type IstioSpec struct {
	Disable            bool   `json:"disable,omitempty"`
	Namespace          string `json:"istio,omitempty"`
	Version            string `json:"version,omitempty"`
	Repo               string `json:"repo,omitempty"`
	IngressGatewayName string `json:"ingressGatewayName,omitempty"`

	Base    BaseSpec    `json:"base,omitempty"`
	Istiod  IstiodSpec  `json:"istiod,omitempty"`
	CNI     CNISpec     `json:"cni,omitempty"`
	Ztunnel ZtunnelSpec `json:"ztunnel,omitempty"`
}

type BaseSpec struct {
	Values string `json:"values,omitempty"`
}

type IstiodSpec struct {
	Values string `json:"values,omitempty"`
}

type CNISpec struct {
	Values string `json:"values,omitempty"`
}

type ZtunnelSpec struct {
	Values string `json:"values,omitempty"`
}

type ServerSpec struct {
	Disable bool   `json:"disable,omitempty"`
	Image   string `json:"image,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

type MDNSSpec struct {
	Disable bool   `json:"disable,omitempty"`
	Image   string `json:"image,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

type TunnelSpec struct {
	Disable bool   `json:"disable,omitempty"`
	Image   string `json:"image,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

type DaemonSpec struct {
	Disable bool   `json:"disable,omitempty"`
	Image   string `json:"image,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

type SettingsSpec struct {
	// AutoUpdateApps (default: true)
	AutoUpdateApps bool `json:"autoUpdateApps,omitempty"`
	// AutoUpdateSystem (default: true)
	AutoUpdateSystem bool `json:"autoUpdateSystem,omitempty"`
	// Hostname defines the base hostname for the install (default: home-cloud.local)
	Hostname string `json:"hostname,omitempty"`
}

type ImageVersion struct {
	Image string
	Tag   string
}

// InstallStatus defines the observed state of Install
type InstallStatus struct {
	Version    string           `json:"version,omitempty"`
	GatewayAPI GatewayAPIStatus `json:"gatewayApi,omitempty"`
	Istio      IstioStatus      `json:"istio,omitempty"`
	Server     ServerStatus     `json:"server,omitempty"`
	Tunnel     TunnelStatus     `json:"tunnel,omitempty"`
	MDNS       MDNSStatus       `json:"mdns,omitempty"`
	Daemon     DaemonStatus     `json:"daemon,omitempty"`
}

type GatewayAPIStatus struct {
	URL     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`
}

type IstioStatus struct {
	Version string `json:"version,omitempty"`
	Repo    string `json:"repo,omitempty"`
}

type ServerStatus struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type MDNSStatus struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type TunnelStatus struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type DaemonStatus struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Install is the Schema for the installs API
type Install struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstallSpec   `json:"spec,omitempty"`
	Status InstallStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InstallList contains a list of Install
type InstallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Install `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Install{}, &InstallList{})
}
