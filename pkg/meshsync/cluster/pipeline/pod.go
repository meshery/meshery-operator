package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Pod will implement step interface for Pods
type Pod struct {
	pipeline.StepContext
	client *discovery.Client
	broker broker.Handler
}

// NewPod - constructor
func NewPod(client *discovery.Client, broker broker.Handler) *Pod {
	return &Pod{
		client: client,
		broker: broker,
	}
}

// Exec - step interface
func (p *Pod) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Pod Discovery Started")

	// get all namespaces
	namespaces := NamespaceName

	for _, namespace := range namespaces {
		// get Pods
		pods, err := p.client.ListPods(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// processing
		for _, pod := range pods {
			// publishing discovered pod
			err := p.broker.Publish(Subject, broker.Message{
				Type:   "Pod",
				Object: pod,
			})
			if err != nil {
				log.Printf("Error publishing pod named %s", pod.Name)
			} else {
				log.Printf("Published pod named %s", pod.Name)
			}
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
