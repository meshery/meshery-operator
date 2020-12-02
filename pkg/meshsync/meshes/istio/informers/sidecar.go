package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) SidecarInformer() cache.SharedIndexInformer {
	// get informer
	SidecarInformer := i.client.GetSidecarInformer().Informer()

	// register event handlers
	SidecarInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				Sidecar := obj.(*v1beta1.Sidecar)
				log.Printf("Sidecar Named: %s - added", Sidecar.Name)
				i.broker.Publish(Subject, broker.Message{
					Type:   "Sidecar",
					Object: Sidecar,
				})
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				Sidecar := new.(*v1beta1.Sidecar)
				log.Printf("Sidecar Named: %s - updated", Sidecar.Name)
				i.broker.Publish(Subject, broker.Message{
					Type:   "Sidecar",
					Object: Sidecar,
				})
			},
			DeleteFunc: func(obj interface{}) {
				Sidecar := obj.(*v1beta1.Sidecar)
				log.Printf("Sidecar Named: %s - deleted", Sidecar.Name)
				i.broker.Publish(Subject, broker.Message{
					Type:   "Sidecar",
					Object: Sidecar,
				})
			},
		},
	)

	return SidecarInformer
}
