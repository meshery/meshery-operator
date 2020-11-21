package discovery

import (
	"istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client will implement discovery functions for kubernetes resources
type Client struct {
	kubeClientset  *kubernetes.Clientset
	istioClientSet *versioned.Clientset
}

// NewKubeClientForConfig constructor
func NewClient(config *rest.Config) (*Client, error) {
	kclientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	iclientSet, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		kubeClientset:  kclientset,
		istioClientSet: iclientSet,
	}, nil
}
