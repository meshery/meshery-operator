package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	kubeconfig *string
	cs         *kubernetes.Clientset
	cfg        *rest.Config
}

func NewClient(kubeconfig *string) (*Client, error) {
	cfg, err := RESTConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	cs, err := ClientsetForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		kubeconfig: kubeconfig,
		cs:         cs,
		cfg:        cfg,
	}, nil
}

func (kcli *Client) Clientset() *kubernetes.Clientset {
	return kcli.cs
}

func (kcli *Client) RESTConfig() *rest.Config {
	return kcli.cfg
}

func (kcli *Client) Kubeconfig() string {
	return *kcli.kubeconfig
}

// Clientset is a helper to return a kubernetes Clientset pointer
// using the kubeconfig values (InCluster = nil, OutOfCluster = <location of kubeconfig>)
func Clientset(kubeconfig *string) (*kubernetes.Clientset, error) {
	config, err := RESTConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func ClientsetForConfig(cfg *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(cfg)
}

// Config is a helper to create and return the kubernetes REST config
// using the kubeconfig values (InCluster = nil, OutOfCluster = <location of kubeconfig>)
func RESTConfig(kubeconfig *string) (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", *kubeconfig)
}
