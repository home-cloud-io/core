package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DiskSpec struct {
	// The node in the cluster the disk is mounted to
	// +required
	Node string `json:"node,omitempty"`
	// valid types: ssd, hdd, nvme
	// +required
	Type string `json:"type,omitempty"`
	// Not necessarily the device name of the disk (e.g. sda) but instead the *stable*
	// alias of the drive (for Talos: the name of the UserVolume?)
	// +required
	Name string `json:"name,omitempty"`
	// whether the operating system is installed to this disk
	// usually do not want to use it also as an application disk
	// +required
	SystemDisk bool `json:"systemDisk,omitempty"`

	Details DiskDetails `json:"details"`
}

type DiskDetails struct {
	// e.g. /dev/sda (keep in mind that this is *not* a stable value)
	DevicePath string `json:"deviceName,omitempty"`
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
