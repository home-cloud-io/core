package k8sclient

import (
	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClient(logger chassis.Logger) *kubernetes.Clientset {
	// NOTE: this will attempt first to build the config from the path given in the draft config and will
	// fallback on the in-cluster config if no path is given
	config := chassis.GetConfig()
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.GetString("server.k8s.master_url"), config.GetString("server.k8s.config_path"))
	if err != nil {
		logger.WithError(err).Panic("failed to build kube config")
	}

	k8sClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		logger.WithError(err).Panic("failed to create kube client")
	}

	return k8sClient
}
