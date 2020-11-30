package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Cluster) NamespaceInformer() cache.SharedIndexInformer {
	// get informer
	namespaceInformer := c.client.GetNamespaceInformer().Informer()

	// register event handlers
	namespaceInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				Namespace := obj.(*v1.Namespace)
				log.Printf("Namespace Named: %s - added", Namespace.Name)
				c.broker.Publish("cluster", broker.Message{
					Type:   "Namespace",
					Object: Namespace,
				})
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				Namespace := new.(*v1.Namespace)
				log.Printf("Namespace Named: %s - updated", Namespace.Name)
				c.broker.Publish("cluster", broker.Message{
					Type:   "Namespace",
					Object: Namespace,
				})
			},
			DeleteFunc: func(obj interface{}) {
				Namespace := obj.(*v1.Namespace)
				log.Printf("Namespace Named: %s - deleted", Namespace.Name)
				c.broker.Publish("cluster", broker.Message{
					Type:   "Namespace",
					Object: Namespace,
				})
			},
		},
	)

	return namespaceInformer
}
