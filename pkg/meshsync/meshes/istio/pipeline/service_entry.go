package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// ServiceEntry will implement step interface for ServiceEntries
type ServiceEntry struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
	broker broker.Handler
}

// NewServiceEntry - constructor
func NewServiceEntry(client *discovery.Client, broker broker.Handler) *ServiceEntry {
	return &ServiceEntry{
		client: client,
		broker: broker,
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

		// processing
		for _, serviceEntry := range serviceEntries {
			// publishing discovered serviceEntry
			err := se.broker.Publish(Subject, broker.Message{
				Type:   "ServiceEntry",
				Object: serviceEntry,
			})
			if err != nil {
				log.Printf("Error publishing service entry named %s", serviceEntry.Name)
			} else {
				log.Printf("Published service entry named %s", serviceEntry.Name)
			}
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
