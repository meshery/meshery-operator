package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// RequestAuthenticaton will implement step interface for RequestAuthenticatons
type RequestAuthenticaton struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
}

// NewRequestAuthenticaton - constructor
func NewRequestAuthenticaton(client *discovery.Client) *RequestAuthenticaton {
	return &RequestAuthenticaton{
		client: client,
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

		// process requestAuthentications
		for _, requestAuthentication := range requestAuthentications {
			log.Printf("Discovered RequestAuthentication named %s in namespace %s", requestAuthentication.Name, namespace)
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
