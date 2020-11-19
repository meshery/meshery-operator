package pipeline

import (
	"github.com/myntra/pipeline"
)

var (
	GlobalDiscoveryStage = &pipeline.Stage{
		Name:       "Global-Resource-Discovery",
		Concurrent: true,
		Steps:      []pipeline.Step{},
	}

	LocalDiscoveryStage = &pipeline.Stage{
		Name:       "Local-Resource-Discovery",
		Concurrent: true,
		Steps:      []pipeline.Step{},
	}
)

func (cluster *Cluster) InitializePipeline() *pipeline.Pipeline {
	// Global discovery
	gdstage := GlobalDiscoveryStage
	gdstage.AddStep(cluster)
	gdstage.AddStep(NewNode(cluster.client))
	gdstage.AddStep(NewNamespace(cluster.client))

	// Local discovery
	ldstage := LocalDiscoveryStage
	ldstage.AddStep(NewDeployment(cluster.client))
	ldstage.AddStep(NewPod(cluster.client))

	// Create Pipeline
	clusterPipeline := pipeline.New("Cluster-Pipeline", 1000)
	clusterPipeline.AddStage(gdstage)
	clusterPipeline.AddStage(ldstage)

	return clusterPipeline
}
