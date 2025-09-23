package host

import (
	"context"
	"strings"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"

	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeClient *kubernetes.Clientset

func KubeClient() *kubernetes.Clientset {
	if kubeClient != nil {
		return kubeClient
	}

	config, err := clientcmd.BuildConfigFromFlags("", KubeConfigFile())
	if err != nil {
		panic(err.Error())
	}

	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return kubeClient
}

func SetSystemImage(ctx context.Context, logger chassis.Logger, def *v1.SetSystemImageCommand) error {
	var (
		replacers = []Replacer{
			func(line string) string {
				return strings.ReplaceAll(line, def.CurrentImage, def.RequestedImage)
			},
		}
		systemKubernetesManifests = []string{
			DraftManifestFile(),
			OperatorManifestFile(),
			ServerManifestFile(),
		}
	)

	for _, filename := range systemKubernetesManifests {
		err := LineByLineReplace(filename, replacers)
		if err != nil {
			return err
		}
	}

	return nil
}
