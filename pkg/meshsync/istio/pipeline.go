package istio

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
)

func (istio *Istio) InitializePipeline() (*pipeline.Pipeline, error) {

	// Mesh Discovery Stage
	mdstage := MeshDiscoveryStage
	mdstage.AddStep(istio)

	// Resource Discovery Stage
	rdstage := ResourcesDiscoveryStage
	rdstage.AddStep(NewVirtualService(istio.Client, istio.KubeClient))
	rdstage.AddStep(NewWorkloadEntry(istio.Client, istio.KubeClient))
	rdstage.AddStep(NewSidecar(istio.Client, istio.KubeClient))

	// Create Pipeline
	istioPipeline := IstioPipeline
	istioPipeline.AddStage(mdstage)
	istioPipeline.AddStage(rdstage)

	return istioPipeline, nil
}
