package istio

import (
	commonpipeline "github.com/layer5io/meshery-operator/pkg/common/pipeline"
	"github.com/myntra/pipeline"
)

var (
	IstioPipeline = &pipeline.Pipeline{
		Name: "Istio-Pipeline",
		Stages: []*pipeline.Stage{
			MeshDiscoveryStage,
			ResourcesDiscoveryStage,
		},
	}

	MeshDiscoveryStage = &pipeline.Stage{
		Name:       "Mesh-Discovery",
		Concurrent: false,
		Steps:      []pipeline.Step{},
	}

	ResourcesDiscoveryStage = &pipeline.Stage{
		Name:       "Resource-Discovery",
		Concurrent: true,
		Steps: []pipeline.Step{
			&VirtualService{},
		},
	}
)

func NewIstioPipeline() (*pipeline.Pipeline, error) {
	return commonpipeline.New(commonpipeline.Options{
		Pipeline: IstioPipeline,
		SkipFail: true,
	})
}
