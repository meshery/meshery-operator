package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Cluster) PodInformer() cache.SharedIndexInformer {
	// get informer
	podInformer := c.client.GetPodInformer().Informer()

	// register event handlers
	podInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				Pod := obj.(*v1.Pod)
				log.Printf("Pod Named: %s - added", Pod.Name)
				c.broker.Publish(Subject, broker.Message{
					Type:   "Pod",
					Object: Pod,
				})
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				Pod := new.(*v1.Pod)
				log.Printf("Pod Named: %s - updated", Pod.Name)
				c.broker.Publish(Subject, broker.Message{
					Type:   "Pod",
					Object: Pod,
				})
			},
			DeleteFunc: func(obj interface{}) {
				Pod := obj.(*v1.Pod)
				log.Printf("Pod Named: %s - deleted", Pod.Name)
				c.broker.Publish(Subject, broker.Message{
					Type:   "Pod",
					Object: Pod,
				})
			},
		},
	)

	return podInformer
}
