package istio

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// Gateway will implement step interface for Gateway
type Gateway struct {
	pipeline.StepContext
	// clients
	client     *discovery.Istio
	kubeclient *discovery.Kubernetes
}

func NewGateway(istioClient *discovery.Istio, kubeClient *discovery.Kubernetes) *Gateway {
	return &Gateway{
		client:     istioClient,
		kubeclient: kubeClient,
	}
}

// Exec - step interface
func (g *Gateway) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Gateway Discovery Started")

	// // get all namespaces
	// namespaces, err := g.kubeclient.ListNamespace()
	// if err != nil {
	// 	return &pipeline.Result{
	// 		Error: err,
	// 	}
	// }

	// // gateway for all namespace
	// for _, namespace := range namespaces {
	// 	Gateways, err := g.client.ListGateway(namespace.Name)
	// 	if err != nil {
	// 		return &pipeline.Result{
	// 			Error: err,
	// 		}
	// 	}

	// 	// process Gateways
	// 	for _, Gateway := range Gateways {
	// 		log.Printf("Discovered gateway named %s in namespace %s", Gateway.Name, namespace.Name)
	// 	}
	// // }

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (g *Gateway) Cancel() error {
	g.Status("cancel step")
	return nil
}
