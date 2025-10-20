package v1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WireguardSpec defines the desired state of Wireguard
type WireguardSpec struct {
	// PrivateKeySecret references a Secret which contains the private key for this interface.
	PrivateKeySecret SecretReference `json:"privateKeySecret"`

	// ListenPort specifies an interface's listening port.
	ListenPort int `json:"listenPort"`

	// Name specifies the name of the interface.
	Name string `json:"name"`

	// Address specifies the address of the interface in CIDR notation.
	Address string `json:"address"`

	// Peers specifies a list of peer configurations to apply to an interface.
	Peers []PeerSpec `json:"peers"`
}

type PeerSpec struct {
	// PrivateKeySecret references a Secret which contains the private key for this peer.
	// +optional
	PrivateKeySecret *SecretReference `json:"privateKeySecret,omitempty"`

	// PublicKey specifies the public key of this peer.  PublicKey is a
	// mandatory field for all PeerConfigs.
	PublicKey string `json:"publicKey"`

	// PresharedKey specifies a peer's preshared key configuration, if not nil.
	//
	// Setting to nil will clear the preshared key.
	// +optional
	PresharedKey *SecretReference `json:"presharedKey,omitempty"`

	// Endpoint specifies the endpoint of this peer entry, if not nil.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`

	// PersistentKeepaliveInterval specifies the persistent keepalive interval
	// for this peer, if not nil.
	//
	// A non-nil value of 0 will clear the persistent keepalive interval.
	// +optional
	PersistentKeepaliveInterval *time.Duration `json:"persistentKeepaliveInterval,omitempty"`

	// AllowedIPs specifies a list of allowed IP addresses in CIDR notation.
	// for this peer.
	AllowedIPs []string `json:"allowedIPs"`
}

type SecretReference struct {
	// Name specifies name of the secret.
	Name string `json:"name"`

	// Namespace specifies the namespace of the secret.
	Namespace string `json:"namespace"`

	// DataKey specifies the data key to find the requested value in.
	DataKey string `json:"dataKey"`
}

// WireguardStatus defines the observed state of Wireguard.
type WireguardStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Wireguard resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Wireguard is the Schema for the wireguards API
type Wireguard struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Wireguard
	// +required
	Spec WireguardSpec `json:"spec"`

	// status defines the observed state of Wireguard
	// +optional
	Status WireguardStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// WireguardList contains a list of Wireguard
type WireguardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Wireguard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Wireguard{}, &WireguardList{})
}
