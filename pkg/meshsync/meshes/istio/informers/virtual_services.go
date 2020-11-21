package informers

import (
	"log"

	v1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) VirtualServiceInformer() cache.SharedIndexInformer {
	// get informer
	VirtualServiceInformer := i.client.GetVirtualServiceInformer().Informer()

	// register event handlers
	VirtualServiceInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addVirtualService,
			UpdateFunc: updateVirtualService,
			DeleteFunc: deleteVirtualService,
		},
	)

	return VirtualServiceInformer
}

func deleteVirtualService(obj interface{}) {
	VirtualService := obj.(*v1beta1.VirtualService)
	log.Printf("VirtualService Named: %s - deleted", VirtualService.Name)
}

func addVirtualService(obj interface{}) {
	VirtualService := obj.(*v1beta1.VirtualService)
	log.Printf("VirtualService Named: %s - added", VirtualService.Name)
}

func updateVirtualService(new interface{}, old interface{}) {
	VirtualService := new.(*v1beta1.VirtualService)
	log.Printf("VirtualService Named: %s - updated", VirtualService.Name)
}
