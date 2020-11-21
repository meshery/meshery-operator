package informers

import (
	networkingV1alpha3 "istio.io/client-go/pkg/informers/externalversions/networking/v1alpha3"
	networkingV1beta1 "istio.io/client-go/pkg/informers/externalversions/networking/v1beta1"
	securityV1beta1 "istio.io/client-go/pkg/informers/externalversions/security/v1beta1"
)

// the methods will return an interface that will implement Informer() and Lister() methods
// TODO: do we need listers?
func (c *Client) GetAuthorizationPolicyInformer() securityV1beta1.AuthorizationPolicyInformer {
	return c.istioInformerFactory.Security().V1beta1().AuthorizationPolicies()
}

func (c *Client) GetPeerAuthenticationInformer() securityV1beta1.PeerAuthenticationInformer {
	return c.istioInformerFactory.Security().V1beta1().PeerAuthentications()
}

func (c *Client) GetRequestAuthenticationInformer() securityV1beta1.RequestAuthenticationInformer {
	return c.istioInformerFactory.Security().V1beta1().RequestAuthentications()
}

func (c *Client) GetDestinationRuleInformer() networkingV1beta1.DestinationRuleInformer {
	return c.istioInformerFactory.Networking().V1beta1().DestinationRules()
}

func (c *Client) GetGatewayInformer() networkingV1beta1.GatewayInformer {
	return c.istioInformerFactory.Networking().V1beta1().Gateways()
}

func (c *Client) GetServiceEntryInformer() networkingV1beta1.ServiceEntryInformer {
	return c.istioInformerFactory.Networking().V1beta1().ServiceEntries()
}

func (c *Client) GetSidecarInformer() networkingV1beta1.SidecarInformer {
	return c.istioInformerFactory.Networking().V1beta1().Sidecars()
}

func (c *Client) GetVirtualServiceInformer() networkingV1beta1.VirtualServiceInformer {
	return c.istioInformerFactory.Networking().V1beta1().VirtualServices()
}

func (c *Client) GetWorkloadEntryInformer() networkingV1beta1.WorkloadEntryInformer {
	return c.istioInformerFactory.Networking().V1beta1().WorkloadEntries()
}

func (c *Client) GetEnvoyFilterInformer() networkingV1alpha3.EnvoyFilterInformer {
	return c.istioInformerFactory.Networking().V1alpha3().EnvoyFilters()
}

func (c *Client) GetWorkloadGroupInformer() networkingV1alpha3.WorkloadGroupInformer {
	return c.istioInformerFactory.Networking().V1alpha3().WorkloadGroups()
}
