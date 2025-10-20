package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
)

var (
	NamespaceObjects = func(install *v1.Install) []client.Object {
		return []client.Object{
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: install.Spec.Istio.Namespace,
					Labels: map[string]string{
						"pod-security.kubernetes.io/enforce": "privileged",
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: install.Spec.HomeCloud.Namespace,
					Labels: map[string]string{
						"pod-security.kubernetes.io/enforce": "privileged",
						"istio.io/dataplane-mode":            "ambient",
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: install.Spec.Draft.Namespace,
					Labels: map[string]string{
						"istio.io/dataplane-mode": "ambient",
					},
				},
			},
		}
	}
)
