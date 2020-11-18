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
)

func (cluster *Cluster) InitializePipeline() (*pipeline.Pipeline, error) {

	// Mesh Discovery Stage
	gdstage := GlobalDiscoveryStage
	gdstage.AddStep(cluster)

	// Create Pipeline
	clusterPipeline := ClusterPipeline
	clusterPipeline.AddStage(gdstage)

	return clusterPipeline, nil
}
