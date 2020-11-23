package informers

import (
	"log"

	v1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) EnvoyFilterInformer() cache.SharedIndexInformer {
	// get informer
	EnvoyFilterInformer := i.client.GetEnvoyFilterInformer().Informer()

	// register event handlers
	EnvoyFilterInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addEnvoyFilter,
			UpdateFunc: updateEnvoyFilter,
			DeleteFunc: deleteEnvoyFilter,
		},
	)

	return EnvoyFilterInformer
}

func deleteEnvoyFilter(obj interface{}) {
	EnvoyFilter := obj.(*v1alpha3.EnvoyFilter)
	log.Printf("EnvoyFilter Named: %s - deleted", EnvoyFilter.Name)
}

func addEnvoyFilter(obj interface{}) {
	EnvoyFilter := obj.(*v1alpha3.EnvoyFilter)
	log.Printf("EnvoyFilter Named: %s - added", EnvoyFilter.Name)
}

func updateEnvoyFilter(new interface{}, old interface{}) {
	EnvoyFilter := new.(*v1alpha3.EnvoyFilter)
	log.Printf("EnvoyFilter Named: %s - updated", EnvoyFilter.Name)
}
