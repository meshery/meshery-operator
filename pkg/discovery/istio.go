package discovery

import (
	"context"

	"istio.io/client-go/pkg/apis/networking/v1beta1"
	"istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Istio to be used in resource discovery
type Istio struct {
	clientSet *versioned.Clientset
}

// NewIstioForConfig constructor
func NewIstioClient(config *rest.Config) (*Istio, error) {
	clientSet, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Istio{
		clientSet: clientSet,
	}, nil
}

// VirtualServices will list virtual service for given namespaces
func (c *Istio) ListVirtualServices(namespace string) ([]v1beta1.VirtualService, error) {
	// get client
	virtualServiceList, err := c.clientSet.
		NetworkingV1beta1().
		VirtualServices(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return virtualServiceList.Items, err
}

// Sidecars will list sidecar for given namespaces
func (c *Istio) ListSidecars(namespace string) ([]v1beta1.Sidecar, error) {
	// get client
	SidecarList, err := c.clientSet.
		NetworkingV1beta1().
		Sidecars(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return SidecarList.Items, err
}

// WorkloadEntrys will list sidecar for given namespaces
func (c *Istio) ListWorkloadEntrys(namespace string) ([]v1beta1.WorkloadEntry, error) {
	// get client
	WorkloadEntryList, err := c.clientSet.
		NetworkingV1beta1().
		WorkloadEntries(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return WorkloadEntryList.Items, err
}
