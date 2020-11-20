package main

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	clusterpipeline "github.com/layer5io/meshery-operator/pkg/meshsync/cluster/pipeline"
	istiopipeline "github.com/layer5io/meshery-operator/pkg/meshsync/meshes/istio/pipeline"
	"k8s.io/client-go/rest"
)

// StartDiscovery - run pipelines
func StartDiscovery(config *rest.Config) error {

	// create discovery client
	client, err := discovery.NewClient(config)
	if err != nil {
		log.Printf("Couldnot create client: %s", err)
		return err
	}

	// get and run pipelines
	// cluster pipeline
	clusterPipeline := clusterpipeline.Initialize(client)

	// istio pipeline
	istioPipeline := istiopipeline.Initialize(client)

	// run pipelines
	result := clusterPipeline.Run()
	if result.Error != nil {
		log.Printf("Error executing cluster pipeline: %s", result.Error)
		return err
	}

	result = istioPipeline.Run()
	if result.Error != nil {
		log.Printf("Error executing istio pipeline: %s", result.Error)
		return err
	}

	return nil
}
