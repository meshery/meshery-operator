package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// WorkloadGroup will implement step interface for WorkloadGroups
type WorkloadGroup struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
	broker broker.Handler
}

// NewWorkloadGroup - constructor
func NewWorkloadGroup(client *discovery.Client, broker broker.Handler) *WorkloadGroup {
	return &WorkloadGroup{
		client: client,
		broker: broker,
	}
}

// Exec - step interface
func (wg *WorkloadGroup) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("WorkloadGroup Discovery Started")

	for _, namespace := range Namespaces {
		workloadGroups, err := wg.client.ListWorkloadGroups(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// processing
		for _, workloadGroup := range workloadGroups {
			// publishing discovered workloadGroup
			err := wg.broker.Publish(Subject, broker.Message{
				Type:   "WorkloadGroup",
				Object: workloadGroup,
			})
			if err != nil {
				log.Printf("Error publishing workload group named %s", workloadGroup.Name)
			} else {
				log.Printf("Published workload group named %s", workloadGroup.Name)
			}
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (wg *WorkloadGroup) Cancel() error {
	wg.Status("cancel step")
	return nil
}
