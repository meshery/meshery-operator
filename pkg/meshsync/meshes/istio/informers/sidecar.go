package informers

import (
	"log"

	v1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) SidecarInformer() cache.SharedIndexInformer {
	// get informer
	SidecarInformer := i.client.GetSidecarInformer().Informer()

	// register event handlers
	SidecarInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addSidecar,
			UpdateFunc: updateSidecar,
			DeleteFunc: deleteSidecar,
		},
	)

	return SidecarInformer
}

func deleteSidecar(obj interface{}) {
	Sidecar := obj.(*v1beta1.Sidecar)
	log.Printf("Sidecar Named: %s - deleted", Sidecar.Name)
}

func addSidecar(obj interface{}) {
	Sidecar := obj.(*v1beta1.Sidecar)
	log.Printf("Sidecar Named: %s - added", Sidecar.Name)
}

func updateSidecar(new interface{}, old interface{}) {
	Sidecar := new.(*v1beta1.Sidecar)
	log.Printf("Sidecar Named: %s - updated", Sidecar.Name)
}
