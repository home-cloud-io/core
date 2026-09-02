package apps

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
)

// TODO: think about making this pluggable for different types of PV sources (ie. not just host path)

var (
	storageClassName = "manual"
)

type (
	Disks []DiskItem
	DiskItem struct {
		Name string
	}
)

func (r *AppReconciler) createPersistence(ctx context.Context, p AppPersistence, app *v1.App, namespace string) error {
	var (
		objName = fmt.Sprintf("%s-%s", app.Spec.Release, p.Name)
	)

	diskList := &v1.DiskList{}
	err := r.Client.List(ctx, diskList)
	if err != nil {
		return err
	}

	basePath := ""
	for _, disk := range diskList.Items {
		if disk.Spec.SystemDisk {
			continue
		}
		// TODO: should have user select the disk for each app
		//		for now we'll just take the first one
		basePath = disk.Spec.Details.MountPath
		break
	}
	if basePath == "" {
		return errors.New("failed to find a non-system disk for persistence")
	}
	hostPath := fmt.Sprintf("%s/%s/%s", basePath, namespace, objName)

	quantity, err := resource.ParseQuantity(p.Size)
	if err != nil {
		return err
	}

	err = r.createPersistentVolume(ctx, objName, namespace, quantity, hostPath)
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	// create PVC
	err = r.createPersistentVolumeClaim(ctx, objName, namespace, quantity)
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	return nil
}


func (r *AppReconciler) deletePersistence(ctx context.Context, p AppPersistence, app *v1.App, namespace string) error {
	var (
		err error
		objName = fmt.Sprintf("%s-%s", app.Spec.Release, p.Name)
	)

	err = r.deletePersistentVolumeClaim(ctx, objName, namespace)
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	err = r.deletePersistentVolume(ctx, objName)
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
}

func (r *AppReconciler) createDiskPersistence(ctx context.Context, disk v1.Disk, app *v1.App, namespace string) (claimName string, err error) {
	var (
		objName = fmt.Sprintf("%s-%s", app.Spec.Release, disk.Name)
	)

	err = r.createPersistentVolume(ctx, objName, namespace, disk.Spec.Details.Size, disk.Spec.Details.MountPath)
	if client.IgnoreAlreadyExists(err) != nil {
		return "", err
	}

	err = r.createPersistentVolumeClaim(ctx, objName, namespace, disk.Spec.Details.Size)
	if client.IgnoreAlreadyExists(err) != nil {
		return "", err
	}

	return objName, nil
}

func (r *AppReconciler) deleteDiskPersistence(ctx context.Context, disk v1.Disk, app *v1.App, namespace string) (claimName string, err error) {
	var (
		objName = fmt.Sprintf("%s-%s", app.Spec.Release, disk.Name)
	)

	err = r.deletePersistentVolumeClaim(ctx, objName, namespace)
	if client.IgnoreNotFound(err) != nil {
		return "", err
	}

	err = r.deletePersistentVolume(ctx, objName)
	if client.IgnoreNotFound(err) != nil {
		return "", err
	}

	return objName, nil
}

func (r *AppReconciler) createPersistentVolume(ctx context.Context, name string, namespace string, quantity resource.Quantity, hostPath string) error {
	return r.Client.Create(ctx, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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
				Name:      name,
			},
			// TODO: NodeAffinity
		},
	})
}

func (r *AppReconciler) deletePersistentVolume(ctx context.Context, name string) error {
	return r.Client.Delete(ctx, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"type": "local",
			},
		},
	})
}

func (r *AppReconciler) createPersistentVolumeClaim(ctx context.Context, name string, namespace string, quantity resource.Quantity) error {
	return r.Client.Create(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
}

func (r *AppReconciler) deletePersistentVolumeClaim(ctx context.Context, name string, namespace string) error {
	return r.Client.Delete(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}
