package discovery

import (
	"context"

	"istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualServices will list virtual service for given namespaces
func (c *Client) ListVirtualServices(namespace string) ([]v1beta1.VirtualService, error) {
	// get client
	virtualServiceList, err := c.istioClientSet.
		NetworkingV1beta1().
		VirtualServices(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return virtualServiceList.Items, err
}

// Sidecars will list sidecar for given namespaces
func (c *Client) ListSidecars(namespace string) ([]v1beta1.Sidecar, error) {
	// get client
	SidecarList, err := c.istioClientSet.
		NetworkingV1beta1().
		Sidecars(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return SidecarList.Items, err
}

// WorkloadEntrys will list sidecar for given namespaces
func (c *Client) ListWorkloadEntrys(namespace string) ([]v1beta1.WorkloadEntry, error) {
	// get client
	WorkloadEntryList, err := c.istioClientSet.
		NetworkingV1beta1().
		WorkloadEntries(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return WorkloadEntryList.Items, err
}
