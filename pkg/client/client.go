package client

import (
	"k8s.io/client-go/rest"

	v1alpha1 "github.com/layer5io/meshery-operator/pkg/client/v1alpha1"
)

type Interface interface {
	CoreV1Alpha1() v1alpha1.CoreInterface
}

type Clientset struct {
	corev1alpha1 *v1alpha1.CoreClient
}

// CoreV1Alpha1 retrieves the CoreV1Alpha1Client
func (c *Clientset) CoreV1Alpha1() v1alpha1.CoreInterface {
	return c.corev1alpha1
}

func New(config *rest.Config) (Interface, error) {
	client, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &Clientset{
		corev1alpha1: v1alpha1.New(client),
	}, nil
}
