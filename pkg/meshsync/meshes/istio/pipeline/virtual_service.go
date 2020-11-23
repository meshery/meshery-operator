package pipeline

import (
	"log"

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
}

// NewVirtualService constructor
func NewVirtualService(client *discovery.Client) *VirtualService {
	return &VirtualService{
		client: client,
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

		// process virtualServices
		for _, virtualService := range virtualServices {
			log.Printf("Discovered virtual service named %s in namespace %s", virtualService.Name, namespace)
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
