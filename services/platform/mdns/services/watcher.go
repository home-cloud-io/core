package services

import (
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func NewServicesWatcher(factory informers.SharedInformerFactory, notifyChan chan<- Resource) (*EventHandler, error) {
	servicesInformer := factory.Core().V1().Services().Informer()
	s := &EventHandler{
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
