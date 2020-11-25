package informers

import (
	inf "github.com/layer5io/meshery-operator/pkg/informers"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Initialize will initiate all the informers
func Initialize(client *inf.Client) {
	c := New(client)

	// initiating informers
	go c.NodeInformer().Run(wait.NeverStop)
	go c.NamespaceInformer().Run(wait.NeverStop)
	go c.DeploymentInformer().Run(wait.NeverStop)
	go c.PodInformer().Run(wait.NeverStop)
}
