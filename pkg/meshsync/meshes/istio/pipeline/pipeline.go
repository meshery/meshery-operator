package pipeline

import (
	broker "github.com/layer5io/meshery-operator/pkg/broker"
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

	// Namespaces : need some solution for this
	Namespaces = []string{"default", "istio-system"}
	Subject    = "istio"
)

func Initialize(client *discovery.Client, broker broker.Handler) *pipeline.Pipeline {
	// Mesh Discovery Stage
	mdstage := MeshDiscoveryStage
	mdstage.AddStep(NewIstio(client, broker))

	// Resource Discovery Stage
	rdstage := ResourcesDiscoveryStage
	rdstage.AddStep(NewAuthorizationPolicy(client, broker))
	rdstage.AddStep(NewDestinationRule(client, broker))
	rdstage.AddStep(NewEnvoyFilter(client, broker))
	rdstage.AddStep(NewGateway(client, broker))
	rdstage.AddStep(NewPeerAuthentication(client, broker))
	rdstage.AddStep(NewRequestAuthenticaton(client, broker))
	rdstage.AddStep(NewServiceEntry(client, broker))
	// rdstage.AddStep(NewWorkloadGroup(client, broker))
	rdstage.AddStep(NewVirtualService(client, broker))
	rdstage.AddStep(NewWorkloadEntry(client, broker))
	rdstage.AddStep(NewSidecar(client, broker))

	// Create Pipeline
	istioPipeline := pipeline.New(Name, 1000)
	istioPipeline.AddStage(mdstage)
	istioPipeline.AddStage(rdstage)

	return istioPipeline
}
