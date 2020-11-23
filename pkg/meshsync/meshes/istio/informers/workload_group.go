package informers

import (
	"log"

	v1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) WorkloadGroupInformer() cache.SharedIndexInformer {
	// get informer
	WorkloadGroupInformer := i.client.GetWorkloadGroupInformer().Informer()

	// register event handlers
	WorkloadGroupInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addWorkloadGroup,
			UpdateFunc: updateWorkloadGroup,
			DeleteFunc: deleteWorkloadGroup,
		},
	)

	return WorkloadGroupInformer
}

func deleteWorkloadGroup(obj interface{}) {
	WorkloadGroup := obj.(*v1alpha3.WorkloadGroup)
	log.Printf("WorkloadGroup Named: %s - deleted", WorkloadGroup.Name)
}

func addWorkloadGroup(obj interface{}) {
	WorkloadGroup := obj.(*v1alpha3.WorkloadGroup)
	log.Printf("WorkloadGroup Named: %s - added", WorkloadGroup.Name)
}

func updateWorkloadGroup(new interface{}, old interface{}) {
	WorkloadGroup := new.(*v1alpha3.WorkloadGroup)
	log.Printf("WorkloadGroup Named: %s - updated", WorkloadGroup.Name)
}
