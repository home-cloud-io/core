package kube

import (
	"os"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	opv1 "github.com/home-cloud-io/core/api/crds/v1"
	talos "github.com/home-cloud-io/core/pkg/talos/api"
)

type (
	Client struct {
		Client          crclient.Client
		Clientset       *kubernetes.Clientset
		DiscoveryClient *discovery.DiscoveryClient
	}
)

func NewClient() (client Client, err error) {
	// NOTE: this will attempt first to build the config from the KUBECONFIG env var and then
	// fallback on the in-cluster config if no path is given
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return
	}

	c, err := crclient.New(kubeConfig, crclient.Options{})
	if err != nil {
		return
	} else {
		apiextensionsv1.AddToScheme(c.Scheme())
		opv1.AddToScheme(c.Scheme())
		gwv1.Install(c.Scheme())
		cmv1.AddToScheme(c.Scheme())
		talos.AddToScheme(c.Scheme())
	}

	cs, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return
	}

	dc, err := discovery.NewDiscoveryClientForConfig(kubeConfig)
	if err != nil {
		return
	}

	return Client{
		Client:          c,
		Clientset:       cs,
		DiscoveryClient: dc,
	}, nil
}
