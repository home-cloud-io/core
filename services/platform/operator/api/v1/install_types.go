package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: omitempty redudant?

// InstallSpec defines the desired state of Install
type InstallSpec struct {
	GatewayAPI GatewayAPISpec `json:"gatewayApiSpec,omitempty"`
	Istio      IstioSpec      `json:"istio,omitempty"`
	Draft      DraftSpec      `json:"draft,omitempty"`
	HomeCloud  HomeCloudSpec  `json:"homeCloud,omitempty"`
	Talos      TalosSpec      `json:"talos,omitempty"`

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

type DraftSpec struct {
	Namespace string        `json:"namespace,omitempty"`
	Blueprint BlueprintSpec `json:"blueprint,omitempty"`
}

type BlueprintSpec struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type HomeCloudSpec struct {
	Namespace string     `json:"namespace,omitempty"`
	Hostname  string     `json:"hostname,omitempty"`
	Server    ServerSpec `json:"server,omitempty"`
	Daemon    DaemonSpec `json:"daemon,omitempty"`
}

type ServerSpec struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type DaemonSpec struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type TalosSpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

// InstallStatus defines the observed state of Install
type InstallStatus struct {
	Installed bool `json:"installed"`
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
