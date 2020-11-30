package informers

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Cluster) DeploymentInformer() cache.SharedIndexInformer {
	// get informer
	deploymentInformer := c.client.GetDeploymentInformer().Informer()

	// register event handlers
	deploymentInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				deployment := obj.(*v1.Deployment)
				log.Printf("Deployment Named: %s - added", deployment.Name)
				c.broker.Publish(Subject, broker.Message{
					Type:   "Deployment",
					Object: deployment,
				})
			},
			UpdateFunc: func(new interface{}, old interface{}) {
				deployment := new.(*v1.Deployment)
				log.Printf("Deployment Named: %s - updated", deployment.Name)
				c.broker.Publish(Subject, broker.Message{
					Type:   "Deployment",
					Object: deployment,
				})
			},
			DeleteFunc: func(obj interface{}) {
				deployment := obj.(*v1.Deployment)
				log.Printf("Deployment Named: %s - deleted", deployment.Name)
				c.broker.Publish(Subject, broker.Message{
					Type:   "Deployment",
					Object: deployment,
				})
			},
		},
	)

	return deploymentInformer
}
