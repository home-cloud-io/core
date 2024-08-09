package source

import (
	"fmt"

	"github.com/blake/external-mdns/resource"
	"github.com/steady-bytes/draft/pkg/chassis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// ServiceSource handles adding, updating, or removing mDNS record advertisements
type ServiceSource struct {
	logger         chassis.Logger
	namespace      string
	notifyChan     chan<- resource.Resource
	sharedInformer cache.SharedIndexInformer
}

// Run starts shared informers and waits for the shared informer cache to
// synchronize.
func (s *ServiceSource) Run(stopCh chan struct{}) error {
	s.sharedInformer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, s.sharedInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}
	return nil
}

func (s *ServiceSource) onAdd(obj interface{}) {
	advertiseResource, err := s.buildRecord(obj, resource.Added)
	if err != nil {
		s.logger.Infof("failed to add: %s/%s", advertiseResource.Namespace, advertiseResource.Name)
	}

	if advertiseResource.Namespace != s.namespace {
		return
	}

	if len(advertiseResource.IPs) == 0 {
		return
	}

	s.logger.Infof("adding: %s/%s", advertiseResource.Namespace, advertiseResource.Name)
	s.notifyChan <- advertiseResource
}

func (s *ServiceSource) onDelete(obj interface{}) {
	advertiseResource, err := s.buildRecord(obj, resource.Deleted)
	if err != nil {
		s.logger.Infof("failed to delete: %s/%s", advertiseResource.Namespace, advertiseResource.Name)
	}
	s.logger.Infof("deleting: %s/%s", advertiseResource.Namespace, advertiseResource.Name)
	s.notifyChan <- advertiseResource
}

func (s *ServiceSource) onUpdate(oldObj interface{}, newObj interface{}) {

	oldResource, err1 := s.buildRecord(oldObj, resource.Deleted)
	if err1 != nil {
		s.logger.Infof("Error parsing old service resource: %s", err1)
	}
	s.notifyChan <- oldResource

	newResource, err2 := s.buildRecord(newObj, resource.Added)
	if err2 != nil {
		s.logger.Infof("Error parsing new service resource: %s", err2)
	}

	s.logger.Infof("updating (old): %s/%s", oldResource.Namespace, oldResource.Name)
	s.logger.Infof("updating (new): %s/%s", newResource.Namespace, newResource.Name)

	s.notifyChan <- newResource
}

func (s *ServiceSource) buildRecord(obj interface{}, action resource.Action) (resource.Resource, error) {

	var advertiseObj = resource.Resource{
		SourceType: "service",
		Action:     action,
	}

	service, ok := obj.(*corev1.Service)

	if !ok {
		return advertiseObj, nil
	}

	advertiseObj.Name = service.Name
	advertiseObj.Namespace = service.Namespace
	advertiseObj.IPs = []string{"192.168.1.184"}

	// if service.Spec.Type == "ClusterIP" && s.publishInternal {
	// 	advertiseObj.IPs = append(advertiseObj.IPs, service.Spec.ClusterIP)
	// } else if service.Spec.Type == "LoadBalancer" {
	// 	for _, lb := range service.Status.LoadBalancer.Ingress {
	// 		if lb.IP != "" {
	// 			advertiseObj.IPs = append(advertiseObj.IPs, lb.IP)
	// 		}
	// 	}
	// }

	return advertiseObj, nil
}

// NewServicesWatcher creates an ServiceSource
func NewServicesWatcher(logger chassis.Logger, factory informers.SharedInformerFactory, namespace string, notifyChan chan<- resource.Resource) ServiceSource {
	servicesInformer := factory.Core().V1().Services().Informer()
	s := &ServiceSource{
		logger:         logger,
		namespace:      namespace,
		notifyChan:     notifyChan,
		sharedInformer: servicesInformer,
	}
	servicesInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.onAdd,
		DeleteFunc: s.onDelete,
		UpdateFunc: s.onUpdate,
	})

	return *s
}
