package informers

import (
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Cluster) NodeInformer() cache.SharedIndexInformer {
	// get informer
	nodeInformer := c.client.GetNodeInformer().Informer()

	// register event handlers
	nodeInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addNode,
			UpdateFunc: updateNode,
			DeleteFunc: deleteNode,
		},
	)

	return nodeInformer
}

func deleteNode(obj interface{}) {
	node := obj.(*v1.Node)
	log.Printf("node Named: %s - deleted", node.Name)
}

func addNode(obj interface{}) {
	node := obj.(*v1.Node)
	log.Printf("node Named: %s - added", node.Name)
}

func updateNode(new interface{}, old interface{}) {
	node := new.(*v1.Node)
	log.Printf("node Named: %s - updated", node.Name)
}
