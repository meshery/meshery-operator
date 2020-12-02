package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) RequestAuthenticationInformer() cache.SharedIndexInformer {
	// get informer
	RequestAuthenticationInformer := i.client.GetRequestAuthenticationInformer().Informer()

	// register event handlers
	RequestAuthenticationInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				RequestAuthentication := obj.(*v1beta1.RequestAuthentication)
				log.Printf("RequestAuthentication Named: %s - added", RequestAuthentication.Name)
				i.broker.Publish(Subject, broker.Message{
					Type:   "RequestAuthentication",
					Object: RequestAuthentication,
				})
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				RequestAuthentication := new.(*v1beta1.RequestAuthentication)
				log.Printf("RequestAuthentication Named: %s - updated", RequestAuthentication.Name)
				i.broker.Publish(Subject, broker.Message{
					Type:   "RequestAuthentication",
					Object: RequestAuthentication,
				})
			},
			DeleteFunc: func(obj interface{}) {
				RequestAuthentication := obj.(*v1beta1.RequestAuthentication)
				log.Printf("RequestAuthentication Named: %s - deleted", RequestAuthentication.Name)
				i.broker.Publish(Subject, broker.Message{
					Type:   "RequestAuthentication",
					Object: RequestAuthentication,
				})
			},
		},
	)

	return RequestAuthenticationInformer
}
