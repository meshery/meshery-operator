package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) WorkloadEntryInformer() cache.SharedIndexInformer {
	// get informer
	WorkloadEntryInformer := i.client.GetWorkloadEntryInformer().Informer()

	// register event handlers
	WorkloadEntryInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				WorkloadEntry := obj.(*v1beta1.WorkloadEntry)
				log.Printf("WorkloadEntry Named: %s - added", WorkloadEntry.Name)
				err := i.broker.Publish(Subject, broker.Message{
					Type:   "WorkloadEntry",
					Object: WorkloadEntry,
				})
				if err != nil {
					log.Println("NATS: Error publishing WorkloadEntry")
				}
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				WorkloadEntry := new.(*v1beta1.WorkloadEntry)
				log.Printf("WorkloadEntry Named: %s - updated", WorkloadEntry.Name)
				err := i.broker.Publish(Subject, broker.Message{
					Type:   "WorkloadEntry",
					Object: WorkloadEntry,
				})
				if err != nil {
					log.Println("NATS: Error publishing WorkloadEntry")
				}
			},
			DeleteFunc: func(obj interface{}) {
				WorkloadEntry := obj.(*v1beta1.WorkloadEntry)
				log.Printf("WorkloadEntry Named: %s - deleted", WorkloadEntry.Name)
				err := i.broker.Publish(Subject, broker.Message{
					Type:   "WorkloadEntry",
					Object: WorkloadEntry,
				})
				if err != nil {
					log.Println("NATS: Error publishing WorkloadEntry")
				}
			},
		},
	)

	return WorkloadEntryInformer
}
