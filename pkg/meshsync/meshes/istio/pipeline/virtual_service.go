package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

const (
	VirtualServiceKey = "virtual-service"
)

// VirtualService will implement step interface for VirtualService
type VirtualService struct {
	pipeline.StepContext
	client *discovery.Client
	broker broker.Handler
}

// NewVirtualService constructor
func NewVirtualService(client *discovery.Client, broker broker.Handler) *VirtualService {
	return &VirtualService{
		client: client,
		broker: broker,
	}
}

// Exec - step interface
func (vs *VirtualService) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Virtual Service Discovery Started")

	// virtual service for all namespace
	for _, namespace := range Namespaces {
		virtualServices, err := vs.client.ListVirtualServices(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// processing
		for _, virtualService := range virtualServices {
			// publishing discovered virtualService
			err := vs.broker.Publish(Subject, broker.Message{
				Type:   "VirtualService",
				Object: virtualService,
			})
			if err != nil {
				log.Printf("Error publishing virtual service named %s", virtualService.Name)
			} else {
				log.Printf("Published virtual service named %s", virtualService.Name)
			}
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (vs *VirtualService) Cancel() error {
	vs.Status("cancel step")
	return nil
}
