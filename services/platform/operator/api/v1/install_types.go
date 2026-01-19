package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: omitempty redudant?

// InstallSpec defines the desired state of Install
type InstallSpec struct {
	GatewayAPI GatewayAPISpec `json:"gatewayApiSpec,omitempty"`
	Istio      IstioSpec      `json:"istio,omitempty"`
	Server     ServerSpec     `json:"homeCloud,omitempty"`
	// optional value to run a system daemon service for managing the host
	// we'll only officially support Talos (for now?) but the community could
	// build others (e.g. NixOS)
	// TODO: document API
	Daemon   DaemonSpec   `json:"talos,omitempty"`
	Settings SettingsSpec `json:"settings,omitempty"`

	VolumeMountHostPath string `json:"volumeMountHostPath,omitempty"`
}

type GatewayAPISpec struct {
	Version string `json:"version,omitempty"`
}

type IstioSpec struct {
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
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type DaemonSpec struct {
	Enabled bool   `json:"enabled,omitempty"`
	Image   string `json:"image,omitempty"`
	Tag     string `json:"tag,omitempty"`

	// TODO: will need options for customizing this install outside of Talos
}

type SettingsSpec struct {
	AutoUpdateApps   bool   `json:"autoUpdateApps"`
	AutoUpdateSystem bool   `json:"autoUpdateSystem"`
	Hostname         string `json:"hostname"`
}

type ImageVersion struct {
	Image string
	Tag   string
}

// InstallStatus defines the observed state of Install
type InstallStatus struct {
	Istio  IstioStatus  `json:"istio,omitempty"`
	Server ServerStatus `json:"server,omitempty"`
	Daemon DaemonStatus `json:"daemon,omitempty"`
}

type IstioStatus struct {
	Version string `json:"version,omitempty"`
	Repo    string `json:"repo,omitempty"`
}

type ServerStatus struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type DaemonStatus struct {
	Enabled bool   `json:"enabled,omitempty"`
	Image   string `json:"image,omitempty"`
	Tag     string `json:"tag,omitempty"`
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
