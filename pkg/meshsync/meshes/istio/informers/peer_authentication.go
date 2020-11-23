package informers

import (
	"log"

	v1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) PeerAuthenticationInformer() cache.SharedIndexInformer {
	// get informer
	PeerAuthenticationInformer := i.client.GetPeerAuthenticationInformer().Informer()

	// register event handlers
	PeerAuthenticationInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addPeerAuthentication,
			UpdateFunc: updatePeerAuthentication,
			DeleteFunc: deletePeerAuthentication,
		},
	)

	return PeerAuthenticationInformer
}

func deletePeerAuthentication(obj interface{}) {
	PeerAuthentication := obj.(*v1beta1.PeerAuthentication)
	log.Printf("PeerAuthentication Named: %s - deleted", PeerAuthentication.Name)
}

func addPeerAuthentication(obj interface{}) {
	PeerAuthentication := obj.(*v1beta1.PeerAuthentication)
	log.Printf("PeerAuthentication Named: %s - added", PeerAuthentication.Name)
}

func updatePeerAuthentication(new interface{}, old interface{}) {
	PeerAuthentication := new.(*v1beta1.PeerAuthentication)
	log.Printf("PeerAuthentication Named: %s - updated", PeerAuthentication.Name)
}
