package cluster

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

type Cluster struct {
	pipeline.StepContext
	client *discovery.Client
}

func New(client *discovery.Client) *Cluster {
	return &Cluster{
		client: client,
	}
}

// Exec - step interface
func (c *Cluster) Exec(request *pipeline.Request) *pipeline.Result {
	log.Println("Cluster Discovery Started")

	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (c *Cluster) Cancel() error {
	c.Status("cancel step")
	return nil
}
