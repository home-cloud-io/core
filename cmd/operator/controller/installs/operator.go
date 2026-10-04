package installs

import (
	"context"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
	"github.com/home-cloud-io/core/pkg/install/resources"
)

func reconcileOperator(ctx context.Context, r *InstallReconciler, install *v1.Install) error {
	installed := install.Status.Operator != nil
	err := r.reconcileObjects(ctx, "operator", install.Spec.Operator.Disable, installed, resources.OperatorObjects(install))
	if err != nil {
		return err
	}
	if !install.Spec.Operator.Disable {
		if install.Spec.Operator.Tag != install.Status.Operator.Tag ||
			install.Spec.Operator.Image != install.Status.Operator.Image {
			install.Status.Operator = &v1.OperatorStatus{
				Image: install.Spec.Operator.Image,
				Tag:   install.Spec.Operator.Tag,
			}
			err := r.Status().Update(ctx, install)
			if err != nil {
				return err
			}

			// shutdown if operator has updated so that the new replica can take over
			r.Cancel()
		}
	} else {
		install.Status.Operator = nil
	}

	return nil
}
