package installs

import (
	"context"
	"fmt"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/install/resources"
	"github.com/home-cloud-io/core/pkg/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func reconcileBlocky(ctx context.Context, r *InstallReconciler, install *v1.Install) error {

	installed := install.Status.Blocky != nil
	err := r.reconcileObjects(ctx, "blocky", install.Spec.Blocky.Disable, installed, resources.BlockyObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.Blocky.Disable {
		install.Status.Blocky = &v1.BlockyStatus{
			Image: install.Spec.Blocky.Image,
			Tag:   install.Spec.Blocky.Tag,
		}
	} else {
		install.Status.Blocky = nil
	}

	return nil
}

func reconcileMDNS(ctx context.Context, r *InstallReconciler, install *v1.Install) error {

	installed := install.Status.MDNS != nil
	err := r.reconcileObjects(ctx, "mdns", install.Spec.MDNS.Disable, installed, resources.MDNSObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.MDNS.Disable {
		install.Status.MDNS = &v1.MDNSStatus{
			Image: install.Spec.MDNS.Image,
			Tag:   install.Spec.MDNS.Tag,
		}
	} else {
		install.Status.MDNS = nil
	}

	return nil
}

func reconcileTunnel(ctx context.Context, r *InstallReconciler, install *v1.Install) error {

	installed := install.Status.Tunnel != nil
	err := r.reconcileObjects(ctx, "tunnel", install.Spec.Tunnel.Disable, installed, resources.TunnelObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.Tunnel.Disable {
		install.Status.Tunnel = &v1.TunnelStatus{
			Image: install.Spec.Tunnel.Image,
			Tag:   install.Spec.Tunnel.Tag,
		}
	} else {
		install.Status.Tunnel = nil
	}

	return nil
}

func reconcileDaemon(ctx context.Context, r *InstallReconciler, install *v1.Install) error {

	installed := install.Status.Daemon != nil
	err := r.reconcileObjects(ctx, "daemon", install.Spec.Daemon.Disable, installed, resources.DaemonObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.Daemon.Disable {
		install.Status.Daemon = &v1.DaemonStatus{
			Image: install.Spec.Daemon.Image,
			Tag:   install.Spec.Daemon.Tag,
		}
	} else {
		install.Status.Daemon = nil
	}

	return nil
}

func reconcileGenericDevicePlugin(ctx context.Context, r *InstallReconciler, install *v1.Install) error {

	installed := install.Status.GenericDevicePlugin != nil
	err := r.reconcileObjects(ctx, "generic-device-plugin", install.Spec.GenericDevicePlugin.Disable, installed, resources.GenericDevicePluginObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.GenericDevicePlugin.Disable {
		install.Status.GenericDevicePlugin = &v1.GenericDevicePluginStatus{
			Image: install.Spec.GenericDevicePlugin.Image,
			Tag:   install.Spec.GenericDevicePlugin.Tag,
		}
	} else {
		install.Status.GenericDevicePlugin = nil
	}

	return nil
}

func (r *InstallReconciler) reconcileObjects(ctx context.Context, name string, disable bool, installed bool, objects []client.Object) error {
	l := log.FromContext(ctx)

	if disable {
		// uninstall if disabled and currently installed
		if installed {
			l.Info(fmt.Sprintf("%s is disabled: removing previous installation", name))
			return kube.UninstallResources(ctx, r.Client, objects)
		}
		return nil
	}

	// otherwise create/update
	l.Info(fmt.Sprintf("reconciling %s install", name))
	return kube.InstallResources(ctx, r.Client, objects)
}
