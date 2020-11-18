package discovery

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListNamespace will list namespace items
func (c *Client) ListNamespaces() ([]corev1.Namespace, error) {
	namespaceList, err := c.kubeClientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	return namespaceList.Items, err
}
