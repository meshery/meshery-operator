package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// AuthorizationPolicy will implement step interface for AuthorizationPolicies
type AuthorizationPolicy struct {
	pipeline.StepContext
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

// NewAuthorizationPolicy - constructor
func NewAuthorizationPolicy(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *AuthorizationPolicy {
	return &AuthorizationPolicy{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (ap *AuthorizationPolicy) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("AuthorizationPolicy Discovery Started")
	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (ap *AuthorizationPolicy) Cancel() error {
	ap.Status("cancel step")
	return nil
}
