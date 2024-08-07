package controller

import (
	"context"
	"fmt"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *AppReconciler) createPersistence(ctx context.Context, p AppPersistence, app *v1.App, namespace string) error {
	var (
		objName          = fmt.Sprintf("%s-%s", app.Spec.Release, p.Name)
		storageClassName = "manual"
	)
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
					Path: fmt.Sprintf("/mnt/k8s-pvs/%s", objName),
				},
			},
			ClaimRef: &corev1.ObjectReference{
				Namespace: namespace,
				Name:      objName,
			},
			// TODO: NodeAffinity
		},
	})
	if err != nil {
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
	if err != nil {
		return err
	}

	return nil
}
