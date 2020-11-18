package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Deployment will implement step interface for Deployments
type Deployment struct {
	pipeline.StepContext
	client *discovery.Client
}

// NewDeployment - constructor
func NewDeployment(client *discovery.Client) *Deployment {
	return &Deployment{
		client: client,
	}
}

// Exec - step interface
func (d *Deployment) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Deployment Discovery Started")

	// get all namespaces
	namespaces := []string{"default"}

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
			log.Printf("Discovered Deployment named %s in namespace %s", deployment.Name, namespace)
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
