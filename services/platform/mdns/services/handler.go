package services

import (
	"fmt"
	"os"

	"github.com/steady-bytes/draft/pkg/chassis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// EventHandler handles Service events from Kubernetes and sends channel messages for mDNS updates
type EventHandler struct {
	logger         chassis.Logger
	namespace      string
	notifyChan     chan<- Resource
	sharedInformer cache.SharedIndexInformer
}

// Run starts shared informers and waits for the shared informer cache to
// synchronize.
func (s *EventHandler) Run(stopCh chan struct{}) {
	s.sharedInformer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, s.sharedInformer.HasSynced) {
		s.logger.Error("timed out waiting for caches to sync")
	}
}

func (s *EventHandler) onAdd(obj interface{}) {
	resource, err := s.buildRecord(obj, Added)
	logger := s.logger.WithFields(chassis.Fields{
		"namespace": resource.Namespace,
		"name":      resource.Name,
	})
	if err != nil {
		logger.WithError(err).WithFields(chassis.Fields{
			"namespace": resource.Namespace,
			"name":      resource.Name,
		}).Error("failed to build record")
	}
	if resource == nil {
		return
	}

	if resource.Namespace != s.namespace {
		logger.WithFields(chassis.Fields{
			"namespace": resource.Namespace,
			"name":      resource.Name,
		}).Debug("ignoring service not in selected namespace")
		return
	}

	logger.Info("adding record")
	s.notifyChan <- *resource
}

func (s *EventHandler) onDelete(obj interface{}) {
	resource, err := s.buildRecord(obj, Deleted)
	logger := s.logger.WithFields(chassis.Fields{
		"namespace": resource.Namespace,
		"name":      resource.Name,
	})
	if err != nil {
		logger.WithError(err).WithFields(chassis.Fields{
			"namespace": resource.Namespace,
			"name":      resource.Name,
		}).Error("failed to build record")
	}
	if resource == nil {
		return
	}

	s.logger.Info("deleting record")
	s.notifyChan <- *resource
}

func (s *EventHandler) onUpdate(oldObj interface{}, newObj interface{}) {

	oldResource, err := s.buildRecord(oldObj, Deleted)
	logger := s.logger.WithFields(chassis.Fields{
		"namespace": oldResource.Namespace,
		"name":      oldResource.Name,
	})
	if err != nil {
		logger.WithError(err).WithFields(chassis.Fields{
			"namespace": oldResource.Namespace,
			"name":      oldResource.Name,
		}).Error("failed to build old record")
	}
	if oldResource != nil {
		s.logger.Info("deleting old record")
		s.notifyChan <- *oldResource
	}

	newResource, err := s.buildRecord(newObj, Added)
	if err != nil {
		logger.WithError(err).WithFields(chassis.Fields{
			"namespace": newResource.Namespace,
			"name":      newResource.Name,
		}).Error("failed to build new record")
	}
	if newResource != nil {
		logger.Info("adding new record")
		s.notifyChan <- *newResource
	}
}

func (h *EventHandler) buildRecord(obj interface{}, action Action) (*Resource, error) {

	service, ok := obj.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("failed to convert object to corev1.Service")
	}
	logger := h.logger.WithFields(chassis.Fields{
		"namespace": service.Namespace,
		"name":      service.Name,
		"address": service.Spec.ExternalName,
	})

	resource := &Resource{
		Action: action,
		Name: service.Name,
		Namespace: service.Namespace,
		IP: service.Spec.ExternalName,
	}

	// only support ExternalName type
	if service.Spec.Type != corev1.ServiceTypeExternalName {
		logger.Debug("ignoring service not in selected namespace")
		return nil, nil
	}


	if resource.IP == "" {
		return resource, fmt.Errorf("service must contain an ExternalName")
	}

	// ignore services that don't match the host IP
	if resource.IP != os.Getenv("HOST_IP") {
		logger.Debug("ignoring service with IP not matching host IP")
		return nil, nil
	}

	return resource, nil
}
