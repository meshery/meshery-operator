package informers

import (
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
)

type Istio struct {
	client *discovery.Client
}

func New(client *discovery.Client) *Istio {
	return &Istio{
		client: client,
	}
}
