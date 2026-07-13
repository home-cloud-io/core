package apps

import (
	"context"
	"fmt"

	"dario.cat/mergo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/install/resources"
)

// TODO: think about making this pluggable for different types of PV sources (ie. not just host path)

const (
	// if daemon is disabled, the user is responsible for creating this hostPath so that
	// Home Cloud can provision PersistentVolumes against it
	DefaultHostPath = "/mnt/home-cloud"
)

func (r *AppReconciler) createPersistence(ctx context.Context, p AppPersistence, app *v1.App, namespace string) error {
	var (
		objName          = fmt.Sprintf("%s-%s", app.Spec.Release, p.Name)
		storageClassName = "manual"
	)

	// get current install config
	install := &v1.Install{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      "install",
		Namespace: "home-cloud-system",
	}, install)
	if err != nil {
		return err
	}

	// set defaults: any values set on the resource will override the defaults
	err = mergo.Merge(install, resources.DefaultInstall)
	if err != nil {
		return err
	}

	// default if no daemon
	hostPath := fmt.Sprintf("%s/%s", DefaultHostPath, objName)

	// if daemon is enabled, get the path before creating PV/PVC
	if !install.Spec.Daemon.Disable {
		// TODO: get the path/disk/UserVolume to use
		// - should use some logic to optimize placement for multi-disk installs?
		// - or just have the user select the disk during install?
		// - I think with this new method we could technically move apps pretty easily between disks
		volumeName := "apps1"
		hostPath = fmt.Sprintf("%s/%s/%s/%s", "/var/mnt", volumeName, namespace, objName)
	}

	quantity, err := resource.ParseQuantity(p.Size)
	if err != nil {
		return err
	}

	// create PV
	err = r.Client.Create(ctx, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: objName,
			Labels: map[string]string{
				"type": "local",
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: storageClassName,
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: quantity,
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: hostPath,
				},
			},
			ClaimRef: &corev1.ObjectReference{
				Namespace: namespace,
				Name:      objName,
			},
			// TODO: NodeAffinity
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	// create PVC
	err = r.Client.Create(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objName,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	return nil
}
