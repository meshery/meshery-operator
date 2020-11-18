package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// WorkloadGroup will implement step interface for WorkloadGroups
type WorkloadGroup struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
}

// NewWorkloadGroup - constructor
func NewWorkloadGroup(client *discovery.Client) *WorkloadGroup {
	return &WorkloadGroup{
		client: client,
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

		// process WorkloadGroups
		for _, workloadGroup := range workloadGroups {
			log.Printf("Discovered Workload  Group named %s in namespace %s", workloadGroup.Name, namespace)
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
