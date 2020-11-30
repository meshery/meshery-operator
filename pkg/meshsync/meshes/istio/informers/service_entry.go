package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) ServiceEntryInformer() cache.SharedIndexInformer {
	// get informer
	ServiceEntryInformer := i.client.GetServiceEntryInformer().Informer()

	// register event handlers
	ServiceEntryInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ServiceEntry := obj.(*v1beta1.ServiceEntry)
				log.Printf("ServiceEntry Named: %s - added", ServiceEntry.Name)
				i.broker.Publish(Subject, broker.Message{
					Type:   "ServiceEntry",
					Object: ServiceEntry,
				})
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				ServiceEntry := new.(*v1beta1.ServiceEntry)
				log.Printf("ServiceEntry Named: %s - updated", ServiceEntry.Name)
				i.broker.Publish(Subject, broker.Message{
					Type:   "ServiceEntry",
					Object: ServiceEntry,
				})
			},
			DeleteFunc: func(obj interface{}) {
				ServiceEntry := obj.(*v1beta1.ServiceEntry)
				log.Printf("ServiceEntry Named: %s - deleted", ServiceEntry.Name)
				i.broker.Publish(Subject, broker.Message{
					Type:   "ServiceEntry",
					Object: ServiceEntry,
				})
			},
		},
	)

	return ServiceEntryInformer
}
