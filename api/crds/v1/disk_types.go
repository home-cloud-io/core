package v1

import (
	"slices"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DiskSpec struct {
	// A user-specified alias that can be used for display purposes
	Alias string `json:"alias,omitempty"`
	// The node in the cluster the disk is mounted to
	// +required
	Node string `json:"node,omitempty"`
	// valid types: ssd, hdd, nvme
	// +required
	Type string `json:"type,omitempty"`
	// A *stable* id of the disk (e.g. /disk/by-path, wwn, serial)
	// +required
	Identifier string `json:"identifier,omitempty"`
	// whether the operating system is installed to this disk
	// usually do not want to use it also as an application disk
	// +required
	SystemDisk bool `json:"systemDisk"`

	Details DiskDetails `json:"details"`
}

type DiskDetails struct {
	// e.g. /dev/sda (keep in mind that this is *not* a stable value)
	DevicePath string `json:"devicePath,omitempty"`
	// e.g. /mnt/my-disk (for Talos: /var/mnt/<alias>)
	MountPath string `json:"mountPath,omitempty"`
	// size in bytes
	Size resource.Quantity `json:"size,omitempty"`

	// below are identifiers which may or may not be set for all disks
	// one or many of these may be used to stably identify a disk for things
	// like selecting a drive for a Talos UserVolume

	Model    string   `json:"model,omitempty"`
	Serial   string   `json:"serial,omitempty"`
	Wwid     string   `json:"wwid,omitempty"`
	Uuid     string   `json:"uuid,omitempty"`
	Symlinks []string `json:"symlinks,omitempty"`
}

type DiskStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Disk is the Schema for the apps API
//
// +kubebuilder:printcolumn:name="Alias",type=string,JSONPath=`.spec.alias`
// +kubebuilder:printcolumn:name="Node",type=string,JSONPath=`.spec.node`
// +kubebuilder:printcolumn:name="System Disk",type=boolean,JSONPath=`.spec.systemDisk`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.details.model`
// +kubebuilder:printcolumn:name="Device Path",type=string,JSONPath=`.spec.details.devicePath`
// +kubebuilder:printcolumn:name="Mount Path",type=string,JSONPath=`.spec.details.mountPath`
type Disk struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiskSpec   `json:"spec,omitempty"`
	Status DiskStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type DiskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Disk `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Disk{}, &DiskList{})
}

// FUNCTIONAL

const (
	AnnotationDiskLoadRequested = "disks.home-cloud.io/load-requested"
	StatusConditionLoaded       = "disks.home-cloud.io/loaded"
)

func (d Disk) Equal(other Disk) bool {
	return d.Spec.Node == other.Spec.Node &&
		d.Spec.Type == other.Spec.Type &&
		d.Spec.Identifier == other.Spec.Identifier &&
		d.Spec.SystemDisk == other.Spec.SystemDisk &&
		d.Spec.Details.DevicePath == other.Spec.Details.DevicePath &&
		d.Spec.Details.MountPath == other.Spec.Details.MountPath &&
		d.Spec.Details.Size.Equal(other.Spec.Details.Size) &&
		d.Spec.Details.Model == other.Spec.Details.Model &&
		d.Spec.Details.Serial == other.Spec.Details.Serial &&
		d.Spec.Details.Wwid == other.Spec.Details.Wwid &&
		d.Spec.Details.Uuid == other.Spec.Details.Uuid &&
		slices.Equal(d.Spec.Details.Symlinks, other.Spec.Details.Symlinks)
}
