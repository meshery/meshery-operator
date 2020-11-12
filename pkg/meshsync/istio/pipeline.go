package istio

import (
	common "github.com/layer5io/meshery-operator/pkg/common"
	client "github.com/layer5io/meshery-operator/pkg/meshsync/istio/client"
	"github.com/myntra/pipeline"
	"k8s.io/client-go/rest"
)

var (
	concurrent = true
	sequential = false
)

// this file will create all the stages
// it will just get the steps and we will arrange them here

// New will return a Pipeline
func New(config *rest.Config, kubeClient *common.KubeClient) (*pipeline.Pipeline, error) {
	// create istio client
	istioClient, err := client.NewIstioClientForConfig(config)
	if err != nil {
		return nil, err
	}

	// new pipeline
	istioPipeline := pipeline.New("istio-discovery", 1000)

	// creating istio specific stages
	// stage-1
	stage1 := pipeline.NewStage("stage-1", sequential, false)
	// creating steps for this stage
	step1 := &VirtualService{
		client:     istioClient,
		kubeclient: kubeClient,
	}
	// adding steps to  the  stage
	stage1.AddStep(step1)

	// adding stages to pipeline
	istioPipeline.AddStage(stage1)

	return istioPipeline, nil
}
