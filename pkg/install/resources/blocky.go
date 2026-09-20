package resources

import (
	"fmt"

	"github.com/steady-bytes/draft/pkg/chassis"
	"go.yaml.in/yaml/v4"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/home-cloud-io/core/api/crds/v1"
)

const (
	HostIPConfigKey = "operator.host_ip"
)

var (
	BlockyObjects = func(install *v1.Install) []client.Object {
		bconfig := BlockyConfig{
			Upstreams: BlockyUpstreams{
				Groups: map[string][]string{
					"default": install.Spec.Settings.Network.DNS.UpstreamServers,
				},
			},
			Ports: BlockyPorts{
				DNS:  53,
				HTTP: 4000,
			},
			Blocking: BlockyBlocking{
				DenyLists: map[string][]string{
					"main": combineDomainsAndSources(install.Spec.Settings.Network.DNS.DenyDomains, install.Spec.Settings.Network.DNS.DenyListSources),
				},
				AllowLists: map[string][]string{
					"main": combineDomainsAndSources(install.Spec.Settings.Network.DNS.AllowDomains, install.Spec.Settings.Network.DNS.AllowListSources),
				},
				ClientGroupsBlock: map[string][]string{
					"default": {
						"main",
					},
				},
			},
			CustomDNS: BlockyCustomDNS{
				CustomTTL: "1h",
				Mapping: map[string]string{
					install.Spec.Settings.Network.Domain: chassis.GetConfig().GetString(HostIPConfigKey),
				},
			},
		}

		byaml, err := yaml.Marshal(bconfig)
		if err != nil {
			// TODO: we don't yet have a pattern for error handling in object functions...
			return make([]client.Object, 0)
		}

		return []client.Object{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "blocky",
					Namespace: install.Namespace,
				},
				Data: map[string]string{
					"config.yml": string(byaml),
				},
			},
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "blocky",
					Namespace: install.Namespace,
					Labels: map[string]string{
						"app": "blocky",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Name:       "tcp",
							Protocol:   corev1.ProtocolTCP,
							NodePort:   53,
							Port:       53,
							TargetPort: intstr.FromInt(53),
						},
						{
							Name:       "udp",
							Protocol:   corev1.ProtocolUDP,
							NodePort:   53,
							Port:       53,
							TargetPort: intstr.FromInt(53),
						},
					},
					Selector: map[string]string{
						"app": "blocky",
					},
				},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "blocky",
					Namespace: install.Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To[int32](1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "blocky",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "blocky",
								// we don't want ztunnel to capture traffic
								"istio.io/dataplane-mode": "none",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "blocky",
									Image: fmt.Sprintf("%s:%s", install.Spec.Blocky.Image, install.Spec.Blocky.Tag),
									Ports: []corev1.ContainerPort{
										{
											Name:          "tcp",
											Protocol:      corev1.ProtocolTCP,
											ContainerPort: 53,
										},
										{
											Name:          "udp",
											Protocol:      corev1.ProtocolUDP,
											ContainerPort: 53,
										},
										{
											Name:          "http",
											Protocol:      corev1.ProtocolTCP,
											ContainerPort: 4000,
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path: "/",
												Port: intstr.FromString("http"),
											},
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path: "/",
												Port: intstr.FromString("http"),
											},
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "config",
											MountPath: "/app/config.yml",
											SubPath:   "config.yml",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{Name: "blocky"},
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}
)

type (
	BlockyConfig struct {
		Upstreams BlockyUpstreams
		Ports     BlockyPorts
		Blocking  BlockyBlocking
		CustomDNS BlockyCustomDNS `yaml:"customDNS"`
	}
	BlockyUpstreams struct {
		Groups map[string][]string
	}
	BlockyPorts struct {
		DNS  int
		HTTP int
	}
	BlockyBlocking struct {
		DenyLists         map[string][]string `yaml:"denylists"`  // intentionally lowercase
		AllowLists        map[string][]string `yaml:"allowlists"` // intentionally lowercase
		ClientGroupsBlock map[string][]string `yaml:"clientGroupsBlock"`
	}
	BlockyCustomDNS struct {
		CustomTTL string `yaml:"customTTL"`
		Mapping   map[string]string
	}
)

func combineDomainsAndSources(domains, sources []string) []string {
	if len(domains) > 0 {
		denyDomains := "# user specified domains"
		for _, v := range domains {
			denyDomains = fmt.Sprintf("%s\n%s", denyDomains, v)
		}
		return append(sources, denyDomains)
	}
	return sources
}
