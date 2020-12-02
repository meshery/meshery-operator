package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// DestinationRule will implement step interface for DestinationRules
type DestinationRule struct {
	pipeline.StepContext
	client *discovery.Client
	broker broker.Handler
}

// NewDestinationRule - constructor
func NewDestinationRule(client *discovery.Client, broker broker.Handler) *DestinationRule {
	return &DestinationRule{
		client: client,
		broker: broker,
	}
}

// Exec - step interface
func (dr *DestinationRule) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("DestinationRule Discovery Started")

	for _, namespace := range Namespaces {
		destinationRules, err := dr.client.ListDestinationRules(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// processing
		for _, destinationRule := range destinationRules {
			// publishing discovered destinationRule
			err := dr.broker.Publish(Subject, broker.Message{
				Type:   "DestinationRule",
				Object: destinationRule,
			})
			if err != nil {
				log.Printf("Error publishing destination rule named %s", destinationRule.Name)
			} else {
				log.Printf("Published destination rule named %s", destinationRule.Name)
			}
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (dr *DestinationRule) Cancel() error {
	dr.Status("cancel step")
	return nil
}
