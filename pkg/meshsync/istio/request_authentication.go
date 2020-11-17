package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// RequestAuthenticaton will implement step interface for RequestAuthenticatons
type RequestAuthenticaton struct {
	pipeline.StepContext
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

// NewRequestAuthenticaton - constructor
func NewRequestAuthenticaton(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *RequestAuthenticaton {
	return &RequestAuthenticaton{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (ra *RequestAuthenticaton) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("RequestAuthenticaton Discovery Started")
	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (ra *RequestAuthenticaton) Cancel() error {
	ra.Status("cancel step")
	return nil
}
