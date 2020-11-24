package informers

import (
	"time"

	"istio.io/client-go/pkg/clientset/versioned"
	istioInformers "istio.io/client-go/pkg/informers/externalversions"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	// informers
	clusterInformerFactory informers.SharedInformerFactory
	istioInformerFactory   istioInformers.SharedInformerFactory
}

func NewClient(config *rest.Config) (*Client, error) {
	kclientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	iclientSet, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	clusterInformerFactory := informers.NewSharedInformerFactory(kclientset, 100*time.Second)
	istioInformerFactory := istioInformers.NewSharedInformerFactory(iclientSet, 100*time.Second)
	return &Client{
		clusterInformerFactory: clusterInformerFactory,
		istioInformerFactory:   istioInformerFactory,
	}, nil
}
