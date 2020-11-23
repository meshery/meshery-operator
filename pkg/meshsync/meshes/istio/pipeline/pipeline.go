package pipeline

import (
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

var (
	Name               = "Istio-Pipeline"
	MeshDiscoveryStage = &pipeline.Stage{
		Name:       "Mesh-Discovery",
		Concurrent: false,
		Steps:      []pipeline.Step{},
	}

	ResourcesDiscoveryStage = &pipeline.Stage{
		Name:       "Resource-Discovery",
		Concurrent: false,
		Steps:      []pipeline.Step{},
	}

	// TODO: need some solution for this
	Namespaces = []string{"default", "istio-system"}
)

func Initialize(client *discovery.Client) *pipeline.Pipeline {

	// Mesh Discovery Stage
	mdstage := MeshDiscoveryStage
	mdstage.AddStep(NewIstio(client))

	// Resource Discovery Stage
	rdstage := ResourcesDiscoveryStage
	rdstage.AddStep(NewVirtualService(client))
	rdstage.AddStep(NewWorkloadEntry(client))
	rdstage.AddStep(NewSidecar(client))

	// Create Pipeline
	istioPipeline := pipeline.New(Name, 1000)
	istioPipeline.AddStage(mdstage)
	istioPipeline.AddStage(rdstage)

	return istioPipeline
}
