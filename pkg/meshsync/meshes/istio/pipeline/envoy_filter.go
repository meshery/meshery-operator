package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// EnvoyFilter will implement step interface for EnvoyFilters
type EnvoyFilter struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
	broker broker.Handler
}

// NewEnvoyFilter - constructor
func NewEnvoyFilter(client *discovery.Client, broker broker.Handler) *EnvoyFilter {
	return &EnvoyFilter{
		client: client,
		broker: broker,
	}
}

// Exec - step interface
func (ef *EnvoyFilter) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("EnvoyFilter Discovery Started")

	for _, namespace := range Namespaces {
		envoyFilters, err := ef.client.ListEnvoyFilters(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// processing
		for _, envoyFilter := range envoyFilters {
			// publishing discovered envoyFilter
			err := ef.broker.Publish(Subject, broker.Message{
				Type:   "EnvoyFilter",
				Object: envoyFilter,
			})
			if err != nil {
				log.Printf("Error publishing envoy filter named %s", envoyFilter.Name)
			} else {
				log.Printf("Published envoy filter named %s", envoyFilter.Name)
			}
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (ef *EnvoyFilter) Cancel() error {
	ef.Status("cancel step")
	return nil
}
