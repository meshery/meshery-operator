package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Deployment will implement step interface for Deployments
type Deployment struct {
	pipeline.StepContext
	client *discovery.Client
	broker broker.Handler
}

// NewDeployment - constructor
func NewDeployment(client *discovery.Client, broker broker.Handler) *Deployment {
	return &Deployment{
		client: client,
		broker: broker,
	}
}

// Exec - step interface
func (d *Deployment) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Deployment Discovery Started")

	// get all namespaces
	namespaces := NamespaceName

	for _, namespace := range namespaces {
		// get Deployments
		deployments, err := d.client.ListDeployments(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// processing
		for _, deployment := range deployments {
			// publishing discovered deployment
			err := d.broker.Publish(Subject, broker.Message{
				Type:   "Deployment",
				Object: deployment,
			})
			if err != nil {
				log.Printf("Error publishing deployment named %s", deployment.Name)
			} else {
				log.Printf("Published deployment named %s", deployment.Name)
			}
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (d *Deployment) Cancel() error {
	d.Status("cancel step")
	return nil
}
