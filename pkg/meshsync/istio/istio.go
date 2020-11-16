package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

type Istio struct {
	pipeline.StepContext
	Client     *discovery.Istio
	KubeClient *discovery.Kubernetes
}

func New(client *discovery.Istio, kubeclient *discovery.Kubernetes) *Istio {
	return &Istio{
		Client:     client,
		KubeClient: kubeclient,
	}
}

// Exec - step interface
func (i *Istio) Exec(request *pipeline.Request) *pipeline.Result {
	log.Println("Istio Discovery Started")

	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (i *Istio) Cancel() error {
	i.Status("cancel step")
	return nil
}
