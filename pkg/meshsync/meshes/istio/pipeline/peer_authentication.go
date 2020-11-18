package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// PeerAuthentication will implement step interface for PeerAuthentications
type PeerAuthentication struct {
	pipeline.StepContext
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

// NewPeerAuthentication - constructor
func NewPeerAuthentication(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *PeerAuthentication {
	return &PeerAuthentication{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (pa *PeerAuthentication) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("PeerAuthentication Discovery Started")
	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (pa *PeerAuthentication) Cancel() error {
	pa.Status("cancel step")
	return nil
}
