package informers

import (
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Cluster) NamespaceInformer() cache.SharedIndexInformer {
	// get informer
	namespaceInformer := c.client.GetNamespaceInformer().Informer()

	// register event handlers
	namespaceInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addNamespace,
			UpdateFunc: updateNamespace,
			DeleteFunc: deleteNamespace,
		},
	)

	return namespaceInformer
}

func deleteNamespace(obj interface{}) {
	namespace := obj.(*v1.Namespace)
	log.Printf("Namespace Named: %s - deleted", namespace.Name)
}

func addNamespace(obj interface{}) {
	namespace := obj.(*v1.Namespace)
	log.Printf("Namespace Named: %s - added", namespace.Name)
}

func updateNamespace(new interface{}, old interface{}) {
	namespace := new.(*v1.Namespace)
	log.Printf("Namespace Named: %s - updated", namespace.Name)
}
