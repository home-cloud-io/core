package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppSpec defines the desired state of an App
type AppSpec struct {
	// Chart is the Helm chart which defines the App.
	Chart string `json:"chart"`
	// Repo is the URL for the chart repository.
	Repo string `json:"repo"`
	// Release is the name of the Helm release of the App.
	Release string `json:"release"`
	// Values optionally defines the values that will be applied to the Chart.
	Values string `json:"values,omitempty"`
}

// AppStatus defines the observed state of an App
type AppStatus struct {
	// Version is the version of the Chart that is currently installed.
	Version string `json:"version"`
	// Values that were used for the current Chart install.
	Values string `json:"values,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// App is the Schema for the apps API
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AppList contains a list of Apps
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}
