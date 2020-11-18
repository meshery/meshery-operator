package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Namespace will implement step interface for Namespaces
type Namespace struct {
	pipeline.StepContext
	client *discovery.Client
}

// NewNamespace - constructor
func NewNamespace(client *discovery.Client) *Namespace {
	return &Namespace{
		client: client,
	}
}

// Exec - step interface
func (n *Namespace) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Namespace Discovery Started")

	// get Namespaces
	namespaces, err := n.client.ListNamespaces()
	if err != nil {
		return &pipeline.Result{
			Error: err,
		}
	}

	// processing
	for _, namespace := range namespaces {
		log.Printf("Discovered namespace named %s", namespace.Name)
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (n *Namespace) Cancel() error {
	n.Status("cancel step")
	return nil
}
