package informers

import (
	inf "github.com/layer5io/meshery-operator/pkg/informers"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Initialize will initiate all the informers
func Initialize(client *inf.Client) error {
	c := New(client)

	// initiating informers
	go c.VirtualServiceInformer().Run(wait.NeverStop)
	go c.SidecarInformer().Run(wait.NeverStop)
	go c.WorkloadEntryInformer().Run(wait.NeverStop)
	go c.AuthorizationPolicyInformer().Run(wait.NeverStop)
	go c.DestinationRuleInformer().Run(wait.NeverStop)
	go c.EnvoyFilterInformer().Run(wait.NeverStop)
	go c.GatewayInformer().Run(wait.NeverStop)
	go c.PeerAuthenticationInformer().Run(wait.NeverStop)
	go c.RequestAuthenticationInformer().Run(wait.NeverStop)
	go c.ServiceEntryInformer().Run(wait.NeverStop)
	// go c.WorkloadGroupInformer().Run(wait.NeverStop)

	return nil
}
