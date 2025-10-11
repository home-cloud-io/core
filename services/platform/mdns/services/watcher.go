package services

import (
	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func NewServicesWatcher(logger chassis.Logger, factory informers.SharedInformerFactory, namespace string, notifyChan chan<- Resource) (*EventHandler, error) {
	servicesInformer := factory.Core().V1().Services().Informer()
	s := &EventHandler{
		logger:         logger,
		namespace:      namespace,
		notifyChan:     notifyChan,
		sharedInformer: servicesInformer,
	}
	_, err := servicesInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.onAdd,
		DeleteFunc: s.onDelete,
		UpdateFunc: s.onUpdate,
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}
