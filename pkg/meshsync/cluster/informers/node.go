package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Cluster) NodeInformer() cache.SharedIndexInformer {
	// get informer
	nodeInformer := c.client.GetNodeInformer().Informer()

	// register event handlers
	nodeInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				Node := obj.(*v1.Node)
				log.Printf("Node Named: %s - added", Node.Name)
				c.broker.Publish(Subject, broker.Message{
					Type:   "Node",
					Object: Node,
				})
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				Node := new.(*v1.Node)
				log.Printf("Node Named: %s - updated", Node.Name)
				c.broker.Publish(Subject, broker.Message{
					Type:   "Node",
					Object: Node,
				})
			},
			DeleteFunc: func(obj interface{}) {
				Node := obj.(*v1.Node)
				log.Printf("Node Named: %s - deleted", Node.Name)
				c.broker.Publish(Subject, broker.Message{
					Type:   "Node",
					Object: Node,
				})
			},
		},
	)

	return nodeInformer
}
