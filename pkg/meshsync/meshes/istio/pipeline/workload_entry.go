package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// WorkloadEntry will implement step interface for WorkloadEntry
type WorkloadEntry struct {
	pipeline.StepContext
	client *discovery.Client
}

func NewWorkloadEntry(client *discovery.Client) *WorkloadEntry {
	return &WorkloadEntry{
		client: client,
	}
}

// Exec - step interface
func (we *WorkloadEntry) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Workload  Entry Discovery Started")

	// // get all namespaces
	// namespaces, err := we.kubeclient.ListNamespace()
	// if err != nil {
	// 	return &pipeline.Result{
	// 		Error: err,
	// 	}
	// }

	// // Workload  Entry for all namespace
	// for _, namespace := range namespaces {
	// 	WorkloadEntries, err := we.client.ListWorkloadEntry(namespace.Name)
	// 	if err != nil {
	// 		return &pipeline.Result{
	// 			Error: err,
	// 		}
	// 	}

	// 	// process WorkloadEntries
	// 	for _, WorkloadEntry := range WorkloadEntries {
	// 		log.Printf("Discovered Workload  Entry named %s in namespace %s", WorkloadEntry.Name, namespace.Name)
	// 	}
	// }

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (we *WorkloadEntry) Cancel() error {
	we.Status("cancel step")
	return nil
}
