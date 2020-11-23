package informers

import (
	informers "github.com/layer5io/meshery-operator/pkg/informers"
)

type Cluster struct {
	client *informers.Client
}

func New(client *informers.Client) *Cluster {
	return &Cluster{
		client: client,
	}
}
