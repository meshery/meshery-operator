package istio

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
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

func NewVirtualService(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *VirtualService {
	return &VirtualService{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (vs *VirtualService) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Virtual Service Discovery Started")

	// // get all namespaces
	// namespaces, err := vs.kubeclient.ListNamespace()
	// if err != nil {
	// 	return &pipeline.Result{
	// 		Error: err,
	// 	}
	// }

	// // virtual service for all namespace
	// for _, namespace := range namespaces {
	// 	virtualServices, err := vs.client.ListVirtualService(namespace.Name)
	// 	if err != nil {
	// 		return &pipeline.Result{
	// 			Error: err,
	// 		}
	// 	}

	// 	// process virtualServices
	// 	for _, virtualService := range virtualServices {
	// 		log.Printf("Discovered virtual service named %s in namespace %s", virtualService.Name, namespace.Name)
	// 	}
	// }

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
