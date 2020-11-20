package pipeline

import (
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

var (
	Name                 = "Cluster-Pipeline"
	GlobalDiscoveryStage = &pipeline.Stage{
		Name:       "Global-Resource-Discovery",
		Concurrent: false,
		Steps:      []pipeline.Step{},
	}

	LocalDiscoveryStage = &pipeline.Stage{
		Name:       "Local-Resource-Discovery",
		Concurrent: false,
		Steps:      []pipeline.Step{},
	}
)

func Initialize(client *discovery.Client) *pipeline.Pipeline {
	// Global discovery
	gdstage := GlobalDiscoveryStage
	gdstage.AddStep(NewCluster(client))
	gdstage.AddStep(NewNode(client))
	gdstage.AddStep(NewNamespace(client))

	// Local discovery
	ldstage := LocalDiscoveryStage
	ldstage.AddStep(NewDeployment(cluster.client))
	ldstage.AddStep(NewPod(cluster.client))

	// Create Pipeline
	clusterPipeline := pipeline.New(Name, 1000)
	clusterPipeline.AddStage(gdstage)
	clusterPipeline.AddStage(ldstage)

	return clusterPipeline
}
