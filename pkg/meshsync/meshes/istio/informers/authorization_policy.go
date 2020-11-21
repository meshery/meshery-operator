package informers

import (
	"log"

	v1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) AuthorizationPolicyInformer() cache.SharedIndexInformer {
	// get informer
	AuthorizationPolicyInformer := i.client.GetAuthorizationPolicyInformer().Informer()

	// register event handlers
	AuthorizationPolicyInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addAuthorizationPolicy,
			UpdateFunc: updateAuthorizationPolicy,
			DeleteFunc: deleteAuthorizationPolicy,
		},
	)

	return AuthorizationPolicyInformer
}

func deleteAuthorizationPolicy(obj interface{}) {
	AuthorizationPolicy := obj.(*v1beta1.AuthorizationPolicy)
	log.Printf("AuthorizationPolicy Named: %s - deleted", AuthorizationPolicy.Name)
}

func addAuthorizationPolicy(obj interface{}) {
	AuthorizationPolicy := obj.(*v1beta1.AuthorizationPolicy)
	log.Printf("AuthorizationPolicy Named: %s - added", AuthorizationPolicy.Name)
}

func updateAuthorizationPolicy(new interface{}, old interface{}) {
	AuthorizationPolicy := new.(*v1beta1.AuthorizationPolicy)
	log.Printf("AuthorizationPolicy Named: %s - updated", AuthorizationPolicy.Name)
}
