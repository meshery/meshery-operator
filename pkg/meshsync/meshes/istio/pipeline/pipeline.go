package pipeline

import (
	"github.com/myntra/pipeline"
)

var (
	IstioPipeline = &pipeline.Pipeline{
		Name:   "Istio-Pipeline",
		Stages: []*pipeline.Stage{},
	}

	MeshDiscoveryStage = &pipeline.Stage{
		Name:       "Mesh-Discovery",
		Concurrent: false,
		Steps:      []pipeline.Step{},
	}

	ResourcesDiscoveryStage = &pipeline.Stage{
		Name:       "Resource-Discovery",
		Concurrent: true,
		Steps:      []pipeline.Step{},
	}

	// TODO: need some solution for this
	Namespaces = []string{"default", "istio-system"}
)

func (istio *Istio) InitializePipeline() (*pipeline.Pipeline, error) {

	// Mesh Discovery Stage
	mdstage := MeshDiscoveryStage
	mdstage.AddStep(istio)

	// Resource Discovery Stage
	rdstage := ResourcesDiscoveryStage
	rdstage.AddStep(NewVirtualService(istio.client))
	rdstage.AddStep(NewWorkloadEntry(istio.client))
	rdstage.AddStep(NewSidecar(istio.client))

	// Create Pipeline
	istioPipeline := IstioPipeline
	istioPipeline.AddStage(mdstage)
	istioPipeline.AddStage(rdstage)

	return istioPipeline, nil
}
