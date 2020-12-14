package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) AuthorizationPolicyInformer() cache.SharedIndexInformer {
	// get informer
	AuthorizationPolicyInformer := i.client.GetAuthorizationPolicyInformer().Informer()

	// register event handlers
	AuthorizationPolicyInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				AuthorizationPolicy := obj.(*v1beta1.AuthorizationPolicy)
				log.Printf("AuthorizationPolicy Named: %s - added", AuthorizationPolicy.Name)
				err := i.broker.Publish(Subject, broker.Message{
					Type:   "AuthorizationPolicy",
					Object: AuthorizationPolicy,
				})
				if err != nil {
					log.Println("NATS: Error publishing AuthorizationPolicy")
				}
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				AuthorizationPolicy := new.(*v1beta1.AuthorizationPolicy)
				log.Printf("AuthorizationPolicy Named: %s - updated", AuthorizationPolicy.Name)
				err := i.broker.Publish(Subject, broker.Message{
					Type:   "AuthorizationPolicy",
					Object: AuthorizationPolicy,
				})
				if err != nil {
					log.Println("NATS: Error publishing AuthorizationPolicy")
				}
			},
			DeleteFunc: func(obj interface{}) {
				AuthorizationPolicy := obj.(*v1beta1.AuthorizationPolicy)
				log.Printf("AuthorizationPolicy Named: %s - deleted", AuthorizationPolicy.Name)
				err := i.broker.Publish(Subject, broker.Message{
					Type:   "AuthorizationPolicy",
					Object: AuthorizationPolicy,
				})
				if err != nil {
					log.Println("NATS: Error publishing AuthorizationPolicy")
				}
			},
		},
	)

	return AuthorizationPolicyInformer
}
