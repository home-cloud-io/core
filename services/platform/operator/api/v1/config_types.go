package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigSpec struct {
	AutoUpdateApps bool `json:"autoUpdateApps"`
	AutoUpdateOS   bool `json:"autoUpdateOS"`
}

type ConfigStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}
