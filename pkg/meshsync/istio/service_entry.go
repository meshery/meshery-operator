package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// ServiceEntry will implement step interface for ServiceEntries
type ServiceEntry struct {
	pipeline.StepContext
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

// NewServiceEntry - constructor
func NewServiceEntry(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *ServiceEntry {
	return &ServiceEntry{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (se *ServiceEntry) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("ServiceEntry Discovery Started")
	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (se *ServiceEntry) Cancel() error {
	se.Status("cancel step")
	return nil
}
