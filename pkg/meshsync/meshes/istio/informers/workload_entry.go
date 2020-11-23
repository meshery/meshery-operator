package informers

import (
	"log"

	v1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) WorkloadEntryInformer() cache.SharedIndexInformer {
	// get informer
	WorkloadEntryInformer := i.client.GetWorkloadEntryInformer().Informer()

	// register event handlers
	WorkloadEntryInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addWorkloadEntry,
			UpdateFunc: updateWorkloadEntry,
			DeleteFunc: deleteWorkloadEntry,
		},
	)

	return WorkloadEntryInformer
}

func deleteWorkloadEntry(obj interface{}) {
	WorkloadEntry := obj.(*v1beta1.WorkloadEntry)
	log.Printf("WorkloadEntry Named: %s - deleted", WorkloadEntry.Name)
}

func addWorkloadEntry(obj interface{}) {
	WorkloadEntry := obj.(*v1beta1.WorkloadEntry)
	log.Printf("WorkloadEntry Named: %s - added", WorkloadEntry.Name)
}

func updateWorkloadEntry(new interface{}, old interface{}) {
	WorkloadEntry := new.(*v1beta1.WorkloadEntry)
	log.Printf("WorkloadEntry Named: %s - updated", WorkloadEntry.Name)
}
