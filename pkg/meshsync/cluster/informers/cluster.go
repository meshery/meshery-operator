package informers

import (
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
)

type Cluster struct {
	client *discovery.Client
}

func New(client *discovery.Client) *Cluster {
	return &Cluster{
		client: client,
	}
}
