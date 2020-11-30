package informers

import (
	broker "github.com/layer5io/meshery-operator/pkg/broker"
	inf "github.com/layer5io/meshery-operator/pkg/informers"
	"k8s.io/apimachinery/pkg/util/wait"
)

var Subject = "cluster"

// Initialize will initiate all the informers
func Initialize(client *inf.Client, broker broker.Broker) {
	c := New(client, broker)

	// initiating informers
	go c.NodeInformer().Run(wait.NeverStop)
	go c.NamespaceInformer().Run(wait.NeverStop)
	go c.DeploymentInformer().Run(wait.NeverStop)
	go c.PodInformer().Run(wait.NeverStop)
}
