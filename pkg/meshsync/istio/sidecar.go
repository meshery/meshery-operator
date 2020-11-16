package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Sidecar will implement step interface for Sidecar
type Sidecar struct {
	pipeline.StepContext
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

func NewSidecar(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *Sidecar {
	return &Sidecar{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (s *Sidecar) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Sidecar Discovery Started")

	// // get all namespaces
	// namespaces, err := s.kubeclient.ListNamespace()
	// if err != nil {
	// 	return &pipeline.Result{
	// 		Error: err,
	// 	}
	// }

	// // virtual service for all namespace
	// for _, namespace := range namespaces {
	// 	Sidecars, err := s.client.ListSidecar(namespace.Name)
	// 	if err != nil {
	// 		return &pipeline.Result{
	// 			Error: err,
	// 		}
	// 	}

	// 	// process Sidecars
	// 	for _, Sidecar := range Sidecars {
	// 		log.Printf("Discovered sidecar named %s in namespace %s", Sidecar.Name, namespace.Name)
	// 	}
	// }

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (s *Sidecar) Cancel() error {
	s.Status("cancel step")
	return nil
}
