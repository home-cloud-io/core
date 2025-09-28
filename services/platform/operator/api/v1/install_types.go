package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// InstallSpec defines the desired state of Install
type InstallSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version is the version of the installation.
	Version string `json:"version"`
}

// InstallStatus defines the observed state of Install
type InstallStatus struct {
	// Version is the version of the Install that is currently installed.
	Version string `json:"version"`
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
