package services

import (
	"fmt"

	"github.com/steady-bytes/draft/pkg/chassis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// EventHandler handles Service events from Kubernetes and sends channel messages for mDNS updates
type EventHandler struct {
	notifyChan     chan<- Resource
	sharedInformer cache.SharedIndexInformer
}

// Run starts shared informers and waits for the shared informer cache to
// synchronize.
func (s *EventHandler) Run(logger chassis.Logger, stopCh chan struct{}) {
	s.sharedInformer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, s.sharedInformer.HasSynced) {
		logger.Error("timed out waiting for caches to sync")
	}
}

func (s *EventHandler) onAdd(obj any) {
	resource, err := s.buildRecord(obj)
	if resource == nil || err != nil {
		return
	}
	resource.Action = Added
	s.notifyChan <- *resource
}

func (s *EventHandler) onDelete(obj any) {
	resource, err := s.buildRecord(obj)
	if resource == nil || err != nil {
		return
	}
	resource.Action = Deleted
	s.notifyChan <- *resource
}

func (s *EventHandler) onUpdate(oldObj any, newObj any) {
	// first delete old record
	resource, err := s.buildRecord(oldObj)
	if resource == nil || err != nil {
		return
	}
	resource.Action = Deleted
	s.notifyChan <- *resource

	// then add new record
	resource, err = s.buildRecord(newObj)
	if resource == nil || err != nil {
		return
	}
	resource.Action = Added
	s.notifyChan <- *resource
}

func (h *EventHandler) buildRecord(obj any) (*Resource, error) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("failed to convert object to corev1.Service")
	}

	value, found := service.Annotations["home-cloud.io/dns"]
	if !found {
		return nil, nil
	}

	return &Resource{
		Name:      service.Name,
		Namespace: service.Namespace,
		Hostname:  value,
	}, nil
}
