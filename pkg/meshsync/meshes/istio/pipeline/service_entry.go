package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// ServiceEntry will implement step interface for ServiceEntries
type ServiceEntry struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
}

// NewServiceEntry - constructor
func NewServiceEntry(client *discovery.Client) *ServiceEntry {
	return &ServiceEntry{
		client: client,
	}
}

// Exec - step interface
func (se *ServiceEntry) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("ServiceEntry Discovery Started")

	for _, namespace := range Namespaces {
		serviceEntries, err := se.client.ListServiceEntries(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// process serviceEntries
		for _, serviceEntry := range serviceEntries {
			log.Printf("Discovered ServiceEntry named %s in namespace %s", serviceEntry.Name, namespace)
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (se *ServiceEntry) Cancel() error {
	se.Status("cancel step")
	return nil
}
