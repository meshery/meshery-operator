package informers

import (
	informers "github.com/layer5io/meshery-operator/pkg/informers"
)

type Istio struct {
	client *informers.Client
}

func New(client *informers.Client) *Istio {
	return &Istio{
		client: client,
	}
}
