package informers

import (
	"log"

	v1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) GatewayInformer() cache.SharedIndexInformer {
	// get informer
	GatewayInformer := i.client.GetGatewayInformer().Informer()

	// register event handlers
	GatewayInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addGateway,
			UpdateFunc: updateGateway,
			DeleteFunc: deleteGateway,
		},
	)

	return GatewayInformer
}

func deleteGateway(obj interface{}) {
	Gateway := obj.(*v1beta1.Gateway)
	log.Printf("Gateway Named: %s - deleted", Gateway.Name)
}

func addGateway(obj interface{}) {
	Gateway := obj.(*v1beta1.Gateway)
	log.Printf("Gateway Named: %s - added", Gateway.Name)
}

func updateGateway(new interface{}, old interface{}) {
	Gateway := new.(*v1beta1.Gateway)
	log.Printf("Gateway Named: %s - updated", Gateway.Name)
}
