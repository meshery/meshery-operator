package discovery

import (
	"context"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListNamespaces will list namespace items
func (c *Client) ListNamespaces() ([]corev1.Namespace, error) {
	namespaceList, err := c.kubeClientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	return namespaceList.Items, err
}

// ListNodes will list Node items
func (c *Client) ListNodes() ([]corev1.Node, error) {
	nodeList, err := c.kubeClientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	return nodeList.Items, err
}

// ListDeployments for given namespace
func (c *Client) ListDeployments(namespace string) ([]appv1.Deployment, error) {
	deploymentList, err := c.kubeClientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	return deploymentList.Items, err
}

// ListPods for given namespace
func (c *Client) ListPods(namespace string) ([]corev1.Pod, error) {
	podList, err := c.kubeClientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	return podList.Items, err
}
