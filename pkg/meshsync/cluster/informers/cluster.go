package informers

import (
	broker "github.com/layer5io/meshery-operator/pkg/broker"
	informers "github.com/layer5io/meshery-operator/pkg/informers"
)

type Cluster struct {
	client *informers.Client
	broker broker.Broker
}

func New(client *informers.Client, broker broker.Broker) *Cluster {
	return &Cluster{
		client: client,
		broker: broker,
	}
}
