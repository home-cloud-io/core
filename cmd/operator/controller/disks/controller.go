package disks

import (
	"context"
	"reflect"
	"strings"
	"time"

	"connectrpc.com/connect"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/cmd/operator/controller/daemon"
	"github.com/home-cloud-io/core/cmd/operator/controller/shared"
)

const (
	StatusConditionLoadRequested = "home-cloud.io/disk-load-requested"
	StatusConditionLoaded        = "home-cloud.io/disk-loaded"
)

// DiskReconciler reconciles a Disk object
type DiskReconciler struct {
	client.Client
	DiscoveryClient *discovery.DiscoveryClient
	Scheme          *runtime.Scheme
	Config          *rest.Config
}

func (r *DiskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling Disk")

	// Get the CRD that triggered reconciliation
	disk := &v1.Disk{}
	err := r.Get(ctx, req.NamespacedName, disk)
	if err != nil {
		if kerrors.IsNotFound(err) {
			l.Info("Disk resource not found. Assuming this means the resource was deleted and so ignoring.")
			return ctrl.Result{}, nil
		}
		l.Info("Failed to get Disk resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		// poll the daemon for disk changes
		RequeueAfter: time.Minute,
	}, r.reconcile(ctx, disk)
}

func (r *DiskReconciler) reconcile(ctx context.Context, disk *v1.Disk) error {
	l := log.FromContext(ctx)

	install, err := shared.GetInstall(ctx, r.Client)
	if err != nil {
		l.Error(err, "failed to get install")
		return err
	}

	// nothing to do if no daemon
	if install.Spec.Daemon.Disable {
		return nil
	}

	// load disk if requested
	if meta.IsStatusConditionTrue(disk.Status.Conditions, StatusConditionLoadRequested) {

		// tell daemon to load disk
		client := daemon.DaemonClient(install.Spec.Daemon.Address)
		loadResp, err := client.LoadDisk(ctx, connect.NewRequest(&dv1.LoadDiskRequest{
			DevicePath: disk.Spec.Details.DevicePath,
			Name:       disk.Name,
		}))
		if err != nil {
			l.Error(err, "failed to load disk")
			return err
		}

		// update disk with mount path
		disk.Spec.Details.MountPath = loadResp.Msg.MountPath
		err = r.Update(ctx, disk)
		if err != nil {
			l.Error(err, "failed to update disk")
			return err
		}

		// update condition
		meta.SetStatusCondition(&disk.Status.Conditions, metav1.Condition{
			Type:    StatusConditionLoadRequested,
			Status:  metav1.ConditionFalse,
			Reason:  "Loaded",
			Message: "",
		})
		meta.SetStatusCondition(&disk.Status.Conditions, metav1.Condition{
			Type:    StatusConditionLoaded,
			Status:  metav1.ConditionTrue,
			Reason:  "Loaded",
			Message: "",
		})
	}

	// TODO: unload?

	l.Info("reconcile complete")
	return r.Status().Update(ctx, disk)
}

func (r *DiskReconciler) poll() {
	t := time.NewTicker(time.Minute)

	for {
		<-t.C
		ctx := context.Background()
		log := log.FromContext(ctx)

		// make sure daemon is enabled
		install, err := shared.GetInstall(ctx, r.Client)
		if err != nil {
			log.Error(err, "failed to get install")
			continue
		}
		if install.Spec.Daemon.Disable {
			continue
		}

		// get disks from daemon (host)
		client := daemon.DaemonClient(install.Spec.Daemon.Address)
		disksResp, err := client.GetDisks(ctx, connect.NewRequest(&dv1.GetDisksRequest{}))
		if err != nil {
			log.Error(err, "failed to get disks from daemon")
			continue
		}

		// list disks currently registered in kube
		kubeDisks := &v1.DiskList{}
		err = r.Client.List(ctx, kubeDisks)
		if err != nil {
			log.Error(err, "failed to get disks from kube")
		}
		// convert to map based on id
		kubeDiskMap := make(map[string]v1.Disk, 0)
		for _, kubeDisk := range kubeDisks.Items {
			kubeDiskMap[kubeDisk.Spec.Identifier] = kubeDisk
		}

		// match disks from daemon with disks in kube (creating as necessary)
		for _, hostDisk := range disksResp.Msg.Disks {
			// find /dev/device/by-id symlink and use that
			// TODO: can probably make this smarter by using other unique values? (e.g. serial, uuid)
			for _, id := range hostDisk.Symlinks {
				if !strings.HasPrefix(id, "/dev/device/by-id") {
					continue
				}

				new := &v1.Disk{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "",
						Namespace: install.Namespace,
					},
					Spec: v1.DiskSpec{
						// TODO: don't hardcode this
						Node:       "home-cloud",
						Type:       convertDiskType(hostDisk.Type),
						Identifier: id,
						SystemDisk: hostDisk.SystemDisk,
						Details: v1.DiskDetails{
							DevicePath: hostDisk.DevicePath,
							// no mount path until loaded
							MountPath: "",
							Size:      *resource.NewQuantity(int64(hostDisk.Size), resource.DecimalSI),
							Model:     hostDisk.Model,
							Serial:    hostDisk.Serial,
							Wwid:      hostDisk.Wwid,
							Uuid:      hostDisk.Uuid,
							Symlinks:  hostDisk.Symlinks,
						},
					},
				}

				// use by-id symlink to check existence
				old, found := kubeDiskMap[id]
				if found {
					if reflect.DeepEqual(new.Spec, old.Spec) {
						err := r.Client.Update(ctx, new)
						if err != nil {
							log.Error(err, "failed to update disk")
							continue
						}
					}
				}
				err := r.Client.Create(ctx, new)
				if err != nil {
					log.Error(err, "failed to create disk")
					continue
				}
			}
		}
	}
}

func convertDiskType(t dv1.DiskType) string {
	switch t {
	case dv1.DiskType_DEVICE_TYPE_HDD:
		return "HDD"
	case dv1.DiskType_DEVICE_TYPE_SSD:
		return "SSD"
	case dv1.DiskType_DEVICE_TYPE_NVME:
		return "NVME"
	case dv1.DiskType_DEVICE_TYPE_UNSPECIFIED:
		return ""
	default:
		return ""
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DiskReconciler) SetupWithManager(mgr ctrl.Manager) error {

	go r.poll()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Disk{}).
		Complete(r)
}
