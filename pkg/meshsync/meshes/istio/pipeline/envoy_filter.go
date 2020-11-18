package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// EnvoyFilter will implement step interface for EnvoyFilters
type EnvoyFilter struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
}

// NewEnvoyFilter - constructor
func NewEnvoyFilter(client *discovery.Client) *EnvoyFilter {
	return &EnvoyFilter{
		client: client,
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
			log.Println("Discovered envoy filter named %s in namespace %s", envoyFilter.Name, namespace)
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
