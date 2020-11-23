package informers

import (
	"log"

	v1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) ServiceEntryInformer() cache.SharedIndexInformer {
	// get informer
	ServiceEntryInformer := i.client.GetServiceEntryInformer().Informer()

	// register event handlers
	ServiceEntryInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addServiceEntry,
			UpdateFunc: updateServiceEntry,
			DeleteFunc: deleteServiceEntry,
		},
	)

	return ServiceEntryInformer
}

func deleteServiceEntry(obj interface{}) {
	ServiceEntry := obj.(*v1beta1.ServiceEntry)
	log.Printf("ServiceEntry Named: %s - deleted", ServiceEntry.Name)
}

func addServiceEntry(obj interface{}) {
	ServiceEntry := obj.(*v1beta1.ServiceEntry)
	log.Printf("ServiceEntry Named: %s - added", ServiceEntry.Name)
}

func updateServiceEntry(new interface{}, old interface{}) {
	ServiceEntry := new.(*v1beta1.ServiceEntry)
	log.Printf("ServiceEntry Named: %s - updated", ServiceEntry.Name)
}
