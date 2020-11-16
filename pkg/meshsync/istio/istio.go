package istio

import (
	"log"

	"github.com/myntra/pipeline"
)

type Istio struct {
	Client     *client.IstioClient
	KubeClient *common.KubeClient
}

func New(client *client.IstioClient, kubeclient *common.KubeClient) (*Istio, error) {
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
