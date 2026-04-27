package controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
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

	install := &v1.Install{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      "install",
		Namespace: "home-cloud-system",
	}, install)
	if err != nil {
		return err
	}

	hostPath := fmt.Sprintf("%s/%s", DefaultHostPath, objName)

	// if daemon is enabled, create volume before creating PV/PVC and use the returned path
	if !install.Spec.Daemon.Disable {
		resp, err := DaemonClient(install.Spec.Daemon.Address).CreateVolume(ctx, connect.NewRequest(&dv1.CreateVolumeRequest{
			Name:    objName,
			MinSize: p.Size,
			// TODO: update App spec to have min/max
			MaxSize: p.Size,
		}))
		if err != nil {
			return err
		}

		hostPath = resp.Msg.Path
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
