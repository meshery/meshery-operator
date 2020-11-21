package informers

import (
	"log"

	v1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) RequestAuthenticationInformer() cache.SharedIndexInformer {
	// get informer
	RequestAuthenticationInformer := i.client.GetRequestAuthenticationInformer().Informer()

	// register event handlers
	RequestAuthenticationInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addRequestAuthentication,
			UpdateFunc: updateRequestAuthentication,
			DeleteFunc: deleteRequestAuthentication,
		},
	)

	return RequestAuthenticationInformer
}

func deleteRequestAuthentication(obj interface{}) {
	RequestAuthentication := obj.(*v1beta1.RequestAuthentication)
	log.Printf("RequestAuthentication Named: %s - deleted", RequestAuthentication.Name)
}

func addRequestAuthentication(obj interface{}) {
	RequestAuthentication := obj.(*v1beta1.RequestAuthentication)
	log.Printf("RequestAuthentication Named: %s - added", RequestAuthentication.Name)
}

func updateRequestAuthentication(new interface{}, old interface{}) {
	RequestAuthentication := new.(*v1beta1.RequestAuthentication)
	log.Printf("RequestAuthentication Named: %s - updated", RequestAuthentication.Name)
}
