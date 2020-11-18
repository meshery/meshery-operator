package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// DestinationRule will implement step interface for DestinationRules
type DestinationRule struct {
	pipeline.StepContext
	client *discovery.Client
}

// NewDestinationRule - constructor
func NewDestinationRule(client *discovery.Client) *DestinationRule {
	return &DestinationRule{
		client: client,
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
			log.Println("Discovered destination rule named %s in namespace %s", destinationRule.Name, namespace)
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
