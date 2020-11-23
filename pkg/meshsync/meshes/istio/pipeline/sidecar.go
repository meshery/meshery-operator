package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Sidecar will implement step interface for Sidecar
type Sidecar struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
}

// NewSidecar - constructor
func NewSidecar(client *discovery.Client) *Sidecar {
	return &Sidecar{
		client: client,
	}
}

// Exec - step interface
func (s *Sidecar) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Sidecar Discovery Started")

	for _, namespace := range Namespaces {
		sidecars, err := s.client.ListSidecars(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// process Sidecars
		for _, sidecar := range sidecars {
			log.Printf("Discovered sidecar named %s in namespace %s", sidecar.Name, namespace)
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (s *Sidecar) Cancel() error {
	s.Status("cancel step")
	return nil
}
