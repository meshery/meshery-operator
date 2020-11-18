package discovery

import (
	"context"

	networkingV1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	networking "istio.io/client-go/pkg/apis/networking/v1beta1"
	security "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListVirtualServices for given namespace
func (c *Client) ListVirtualServices(namespace string) ([]networking.VirtualService, error) {
	// get client
	virtualServiceList, err := c.istioClientSet.
		NetworkingV1beta1().
		VirtualServices(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return virtualServiceList.Items, err
}

// ListSidecars for given namespace
func (c *Client) ListSidecars(namespace string) ([]networking.Sidecar, error) {
	// get client
	SidecarList, err := c.istioClientSet.
		NetworkingV1beta1().
		Sidecars(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return SidecarList.Items, err
}

// ListWorkloadEntries for given namespace
func (c *Client) ListWorkloadEntries(namespace string) ([]networking.WorkloadEntry, error) {
	// get client
	WorkloadEntryList, err := c.istioClientSet.
		NetworkingV1beta1().
		WorkloadEntries(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return WorkloadEntryList.Items, err
}

// ListDestinationRules for given namespace
func (c *Client) ListDestinationRules(namespace string) ([]networking.DestinationRule, error) {
	// get client
	DestinationRuleList, err := c.istioClientSet.
		NetworkingV1beta1().
		DestinationRules(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return DestinationRuleList.Items, err
}

// ListGateways for given namespace
func (c *Client) ListGateways(namespace string) ([]networking.Gateway, error) {
	// get client
	GatewayList, err := c.istioClientSet.
		NetworkingV1beta1().
		Gateways(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return GatewayList.Items, err
}

// ListServiceEntries for given namespace
func (c *Client) ListServiceEntries(namespace string) ([]networking.ServiceEntry, error) {
	// get client
	ServiceEntryList, err := c.istioClientSet.
		NetworkingV1beta1().
		ServiceEntries(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return ServiceEntryList.Items, err
}

// ListEnvoyFilters for given namespace
func (c *Client) ListEnvoyFilters(namespace string) ([]networkingV1alpha3.EnvoyFilter, error) {
	// get client
	EnvoyFilterList, err := c.istioClientSet.
		NetworkingV1alpha3().
		EnvoyFilters(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return EnvoyFilterList.Items, err
}

// ListWorkloadGroups for given namespace
func (c *Client) ListWorkloadGroups(namespace string) ([]networkingV1alpha3.WorkloadGroup, error) {
	// get client
	WorkloadGroupList, err := c.istioClientSet.
		NetworkingV1alpha3().
		WorkloadGroups(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return WorkloadGroupList.Items, err
}

// ListAuthorizationPolicies for given namespace
func (c *Client) ListAuthorizationPolicies(namespace string) ([]security.AuthorizationPolicy, error) {
	// get client
	AuthorizationPolicyList, err := c.istioClientSet.
		SecurityV1beta1().
		AuthorizationPolicies(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return AuthorizationPolicyList.Items, err
}

// ListPeerAuthentications for given namespace
func (c *Client) ListPeerAuthentications(namespace string) ([]security.PeerAuthentication, error) {
	// get client
	PeerAuthenticationList, err := c.istioClientSet.
		SecurityV1beta1().
		PeerAuthentications(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return PeerAuthenticationList.Items, err
}

// ListRequestAuthentications for given namespace
func (c *Client) ListRequestAuthentications(namespace string) ([]security.RequestAuthentication, error) {
	// get client
	RequestAuthenticationList, err := c.istioClientSet.
		SecurityV1beta1().
		RequestAuthentications(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return RequestAuthenticationList.Items, err
}
