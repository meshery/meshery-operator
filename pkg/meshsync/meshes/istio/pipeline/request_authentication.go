package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// RequestAuthenticaton will implement step interface for RequestAuthenticatons
type RequestAuthenticaton struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
	broker broker.Handler
}

// NewRequestAuthenticaton - constructor
func NewRequestAuthenticaton(client *discovery.Client, broker broker.Handler) *RequestAuthenticaton {
	return &RequestAuthenticaton{
		client: client,
		broker: broker,
	}
}

// Exec - step interface
func (ra *RequestAuthenticaton) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("RequestAuthenticaton Discovery Started")

	for _, namespace := range Namespaces {
		requestAuthentications, err := ra.client.ListRequestAuthentications(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// processing
		for _, requestAuthentication := range requestAuthentications {
			// publishing discovered requestAuthentication
			err := ra.broker.Publish(Subject, broker.Message{
				Type:   "RequestAuthentication",
				Object: requestAuthentication,
			})
			if err != nil {
				log.Printf("Error publishing request authentication named %s", requestAuthentication.Name)
			} else {
				log.Printf("Published request authentication named %s", requestAuthentication.Name)
			}
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (ra *RequestAuthenticaton) Cancel() error {
	ra.Status("cancel step")
	return nil
}
