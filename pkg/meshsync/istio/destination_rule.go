package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// DestinationRule will implement step interface for DestinationRules
type DestinationRule struct {
	pipeline.StepContext
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

// NewDestinationRule - constructor
func NewDestinationRule(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *DestinationRule {
	return &DestinationRule{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (dr *DestinationRule) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("DestinationRule Discovery Started")
	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (dr *DestinationRule) Cancel() error {
	dr.Status("cancel step")
	return nil
}
