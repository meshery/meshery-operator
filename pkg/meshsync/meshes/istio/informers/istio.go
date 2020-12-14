package informers

import (
	broker "github.com/layer5io/meshery-operator/pkg/broker"
	informers "github.com/layer5io/meshery-operator/pkg/informers"
)

type Istio struct {
	client *informers.Client
	broker broker.Handler
}

func New(client *informers.Client, broker broker.Handler) *Istio {
	return &Istio{
		client: client,
		broker: broker,
	}
}
