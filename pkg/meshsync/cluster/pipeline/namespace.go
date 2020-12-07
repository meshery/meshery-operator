package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

var NamespaceName []string

// Namespace will implement step interface for Namespaces
type Namespace struct {
	pipeline.StepContext
	client *discovery.Client
	broker broker.Handler
}

// NewNamespace - constructor
func NewNamespace(client *discovery.Client, broker broker.Handler) *Namespace {
	return &Namespace{
		client: client,
		broker: broker,
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
		// publishing discovered namespace
		err := n.broker.Publish(Subject, broker.Message{
			Type:   "Namespace",
			Object: namespace,
		})
		if err != nil {
			log.Printf("Error publishing namespace named %s", namespace.Name)
		} else {
			log.Printf("Published namespace named %s", namespace.Name)
		}

		NamespaceName = append(NamespaceName, namespace.Name)
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
