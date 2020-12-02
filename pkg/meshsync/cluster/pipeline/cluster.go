package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

type Cluster struct {
	pipeline.StepContext
	client *discovery.Client
	broker broker.Handler
}

func NewCluster(client *discovery.Client, broker broker.Handler) *Cluster {
	return &Cluster{
		client: client,
		broker: broker,
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
