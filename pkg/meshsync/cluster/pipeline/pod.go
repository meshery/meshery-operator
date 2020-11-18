package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Pod will implement step interface for Pods
type Pod struct {
	pipeline.StepContext
	client *discovery.Client
}

// NewPod - constructor
func NewPod(client *discovery.Client) *Pod {
	return &Pod{
		client: client,
	}
}

// Exec - step interface
func (p *Pod) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Pod Discovery Started")

	// get all namespaces
	namespaces := []string{"default"}

	for _, namespace := range namespaces {
		// get Pods
		Pods, err := p.client.ListPods(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// processing
		for _, Pod := range Pods {
			log.Printf("Discovered Pod named %s in namespace %s", Pod.Name, namespace)
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (p *Pod) Cancel() error {
	p.Status("cancel step")
	return nil
}
