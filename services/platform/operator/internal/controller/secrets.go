package controller

import (
	"context"

	"github.com/home-cloud-io/core/services/platform/operator/internal/controller/secrets"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *AppReconciler) createSecret(ctx context.Context, s AppSecret, namespace string) error {
	// generate secret for each key
	data := map[string][]byte{}
	for _, k := range s.Keys {
		if k.Length == 0 {
			k.Length = 24
		}
		data[k.Name] = secrets.Generate(k.Length, k.NoSpecialCharacters)
	}

	// create secret on cluster
	err := r.Client.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return err
	}

	return nil
}
