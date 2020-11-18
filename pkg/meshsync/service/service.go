package service

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	clusterPipelinePackage "github.com/layer5io/meshery-operator/pkg/meshsync/cluster/pipeline"
	istioPipelinePackage "github.com/layer5io/meshery-operator/pkg/meshsync/meshes/istio/pipeline"
	"k8s.io/client-go/rest"
)

// StartDiscovery - run pipelines
func StartDiscovery() {
	// get kube config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Couldnot load config: %s", err)
		return
	}

	// create discovery client
	client, err := discovery.NewClient(config)
	if err != nil {
		log.Printf("Couldnot create client: %s", err)
		return
	}

	// get and run pipelines
	// cluster pipeline
	cluster := clusterPipelinePackage.New(client)
	clusterPipeline := cluster.InitializePipeline()

	// istio pipeline
	istio := istioPipelinePackage.New(client)
	istioPipeline := istio.InitializePipeline()

	// run pipelines
	result := clusterPipeline.Run()
	if result.Error != nil {
		log.Printf("Error executing cluster pipeline: %s", result.Error)
		return
	}

	result = istioPipeline.Run()
	if result.Error != nil {
		log.Printf("Error executing istio pipeline: %s", result.Error)
		return
	}
}
