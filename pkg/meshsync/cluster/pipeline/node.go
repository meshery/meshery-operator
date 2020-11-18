package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Node will implement step interface for Nodes
type Node struct {
	pipeline.StepContext
	client *discovery.Client
}

// NewNode - constructor
func NewNode(client *discovery.Client) *Node {
	return &Node{
		client: client,
	}
}

// Exec - step interface
func (n *Node) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Node Discovery Started")

	// get nodes
	nodes, err := n.client.ListNodes()
	if err != nil {
		return &pipeline.Result{
			Error: err,
		}
	}

	// processing
	for _, node := range nodes {
		log.Println("Discovered node named %s", node.Name)
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (n *Node) Cancel() error {
	n.Status("cancel step")
	return nil
}
