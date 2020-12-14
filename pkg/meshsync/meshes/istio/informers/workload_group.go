package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) WorkloadGroupInformer() cache.SharedIndexInformer {
	// get informer
	WorkloadGroupInformer := i.client.GetWorkloadGroupInformer().Informer()

	// register event handlers
	WorkloadGroupInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				WorkloadGroup := obj.(*v1alpha3.WorkloadGroup)
				log.Printf("WorkloadGroup Named: %s - added", WorkloadGroup.Name)
				err := i.broker.Publish(Subject, broker.Message{
					Type:   "WorkloadGroup",
					Object: WorkloadGroup,
				})
				if err != nil {
					log.Println("NATS: Error publishing WorkloadGroup")
				}
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				WorkloadGroup := new.(*v1alpha3.WorkloadGroup)
				log.Printf("WorkloadGroup Named: %s - updated", WorkloadGroup.Name)
				err := i.broker.Publish(Subject, broker.Message{
					Type:   "WorkloadGroup",
					Object: WorkloadGroup,
				})
				if err != nil {
					log.Println("NATS: Error publishing WorkloadGroup")
				}
			},
			DeleteFunc: func(obj interface{}) {
				WorkloadGroup := obj.(*v1alpha3.WorkloadGroup)
				log.Printf("WorkloadGroup Named: %s - deleted", WorkloadGroup.Name)
				err := i.broker.Publish(Subject, broker.Message{
					Type:   "WorkloadGroup",
					Object: WorkloadGroup,
				})
				if err != nil {
					log.Println("NATS: Error publishing WorkloadGroup")
				}
			},
		},
	)

	return WorkloadGroupInformer
}
