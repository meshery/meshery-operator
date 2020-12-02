package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Gateway will implement step interface for Gateway
type Gateway struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
	broker broker.Handler
}

func NewGateway(client *discovery.Client, broker broker.Handler) *Gateway {
	return &Gateway{
		client: client,
		broker: broker,
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

		// processing
		for _, gateway := range gateways {
			// publishing discovered gateway
			err := g.broker.Publish(Subject, broker.Message{
				Type:   "Gateway",
				Object: gateway,
			})
			if err != nil {
				log.Printf("Error publishing gateway named %s", gateway.Name)
			} else {
				log.Printf("Published gateway named %s", gateway.Name)
			}
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
