package pipeline

import (
	"github.com/myntra/pipeline"
)

var (
	ClusterPipeline = &pipeline.Pipeline{
		Name:   "Cluster-Pipeline",
		Stages: []*pipeline.Stage{},
	}

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

func (cluster *Cluster) InitializePipeline() (*pipeline.Pipeline, error) {

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
	clusterPipeline := ClusterPipeline
	clusterPipeline.AddStage(gdstage)

	return clusterPipeline, nil
}
