package informers

import (
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Cluster) PodInformer() cache.SharedIndexInformer {
	// get informer
	podInformer := c.client.GetPodInformer().Informer()

	// register event handlers
	podInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addPod,
			UpdateFunc: updatePod,
			DeleteFunc: deletePod,
		},
	)

	return podInformer
}

func deletePod(obj interface{}) {
	pod := obj.(*v1.Pod)
	log.Printf("Pod Named: %s - deleted", pod.Name)
}

func addPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	log.Printf("Pod Named: %s - added", pod.Name)
}

func updatePod(new interface{}, old interface{}) {
	pod := new.(*v1.Pod)
	log.Printf("Pod Named: %s - updated", pod.Name)
}
