package informers

import (
	"log"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Cluster) DeploymentInformer() cache.SharedIndexInformer {
	// get informer
	deploymentInformer := c.client.GetDeploymentInformer().Informer()

	// register event handlers
	deploymentInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addDeployment,
			UpdateFunc: updateDeployment,
			DeleteFunc: deleteDeployment,
		},
	)

	return deploymentInformer
}

func deleteDeployment(obj interface{}) {
	deployment := obj.(*v1.Deployment)
	log.Printf("Deployment Named: %s - deleted", deployment.Name)
}

func addDeployment(obj interface{}) {
	deployment := obj.(*v1.Deployment)
	log.Printf("Deployment Named: %s - added", deployment.Name)
}

func updateDeployment(new interface{}, old interface{}) {
	deployment := new.(*v1.Deployment)
	log.Printf("Deployment Named: %s - updated", deployment.Name)
}
