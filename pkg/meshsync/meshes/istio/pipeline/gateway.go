package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Gateway will implement step interface for Gateway
type Gateway struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
}

func NewGateway(client *discovery.Client) *Gateway {
	return &Gateway{
		client: client,
	}
}

// Exec - step interface
func (g *Gateway) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Gateway Discovery Started")

	// gateway for all namespace
	for _, namespace := range Namespaces {
		gateways, err := g.client.ListGateways(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// process Gateways
		for _, gateway := range gateways {
			log.Printf("Discovered gateway named %s in namespace %s", gateway.Name, namespace)
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (g *Gateway) Cancel() error {
	g.Status("cancel step")
	return nil
}
