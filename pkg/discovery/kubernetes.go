package discovery

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Kubernetes will implement discovery functions for kubernetes resources
type Kubernetes struct {
	clientset *kubernetes.Clientset
}

// NewKubeClientForConfig constructor
func NewKubernetesClient(config *rest.Config) (*Kubernetes, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		clientset: clientset,
	}, nil
}

// ListNamespace will list namespace items
func (c *Kubernetes) Namespaces() ([]v1.Namespace, error) {
	namespaceList, err := c.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	return namespaceList.Items, err
}
