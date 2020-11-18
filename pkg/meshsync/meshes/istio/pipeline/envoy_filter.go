package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// EnvoyFilter will implement step interface for EnvoyFilters
type EnvoyFilter struct {
	pipeline.StepContext
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

// NewEnvoyFilter - constructor
func NewEnvoyFilter(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *EnvoyFilter {
	return &EnvoyFilter{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (ef *EnvoyFilter) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("EnvoyFilter Discovery Started")
	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (ef *EnvoyFilter) Cancel() error {
	ef.Status("cancel step")
	return nil
}
