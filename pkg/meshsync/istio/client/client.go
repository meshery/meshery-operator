package client

import (
	"context"

	"istio.io/client-go/pkg/apis/networking/v1beta1"
	"istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// IstioClient to be used in resource discovery
type IstioClient struct {
	clientSet *versioned.Clientset
}

// NewIstioClientForConfig constructor
func NewIstioClientForConfig(config *rest.Config) (*IstioClient, error) {
	clientSet, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &IstioClient{
		clientSet: clientSet,
	}, nil
}

// ListVirtualService will list virtual service for given namespaces
func (c *IstioClient) ListVirtualService(namespace string) ([]v1beta1.VirtualService, error) {
	// get client
	virtualServiceList, err := c.clientSet.
		NetworkingV1beta1().
		VirtualServices(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return virtualServiceList.Items, err
}

// ListSidecar will list sidecar for given namespaces
func (c *IstioClient) ListSidecar(namespace string) ([]v1beta1.Sidecar, error) {
	// get client
	SidecarList, err := c.clientSet.
		NetworkingV1beta1().
		Sidecars(namespace).
		List(context.TODO(), metav1.ListOptions{})

	return SidecarList.Items, err
}
