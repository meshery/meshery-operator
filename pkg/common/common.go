package common

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubeClient will implement common functionalities
type KubeClient struct {
	clientset *kubernetes.Clientset
}

// NewKubeClientForConfig constructor
func NewKubeClientForConfig(config *rest.Config) (*KubeClient, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		clientset: clientset,
	}, nil
}

// ListNamespace will list namespace items
func (c *KubeClient) ListNamespace() ([]v1.Namespace, error) {
	namespaceList, err := c.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	return namespaceList.Items, err
}
