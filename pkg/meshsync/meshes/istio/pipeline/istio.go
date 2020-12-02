package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

type Istio struct {
	pipeline.StepContext
	client *discovery.Client
	broker broker.Handler
}

func NewIstio(client *discovery.Client, broker broker.Handler) *Istio {
	return &Istio{
		client: client,
		broker: broker,
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
