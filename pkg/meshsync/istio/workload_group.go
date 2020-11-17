package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// WorkloadGroup will implement step interface for WorkloadGroups
type WorkloadGroup struct {
	pipeline.StepContext
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

// NewWorkloadGroup - constructor
func NewWorkloadGroup(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *WorkloadGroup {
	return &WorkloadGroup{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (wg *WorkloadGroup) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("WorkloadGroup Discovery Started")
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
