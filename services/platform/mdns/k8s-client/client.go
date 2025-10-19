package k8sclient

import (
	"os"

	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClient(logger chassis.Logger) *kubernetes.Clientset {
	// NOTE: this will attempt first to build the config from the KUBECONFIG env var and then
	// fallback on the in-cluster config if no path is given
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		logger.WithError(err).Error("failed to read kube config")
		panic(err)
	}

	k8sClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		logger.WithError(err).Panic("failed to create kube client")
	}

	return k8sClient
}
