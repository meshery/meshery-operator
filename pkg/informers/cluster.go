package informers

import (
	appsV1 "k8s.io/client-go/informers/apps/v1"
	coreV1 "k8s.io/client-go/informers/core/v1"
)

// the methods will return an interface that will implement Informer() and Lister() methods
// TODO: do we need listers?
func (c *Client) GetNodeInformer() coreV1.NodeInformer {
	return c.clusterInformerFactory.Core().V1().Nodes()
}

func (c *Client) GetNamespaceInformer() coreV1.NamespaceInformer {
	return c.clusterInformerFactory.Core().V1().Namespaces()
}

func (c *Client) GetDeploymentInformer() appsV1.DeploymentInformer {
	return c.clusterInformerFactory.Apps().V1().Deployments()
}

func (c *Client) GetPodInformer() coreV1.PodInformer {
	return c.clusterInformerFactory.Core().V1().Pods()
}
