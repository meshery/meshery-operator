package main

import (
	"fmt"
	"log"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	inf "github.com/layer5io/meshery-operator/pkg/informers"
	"github.com/layer5io/meshery-operator/pkg/meshsync/cluster"
	"github.com/layer5io/meshery-operator/pkg/meshsync/meshes/istio"

	appsv1 "k8s.io/api/apps/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

func GetResource() runtime.Object {
	fmt.Println("Getting meshsync resource")
	return &appsv1.Deployment{}
}

func CreateSyncController(m *mesheryv1alpha1.MeshSync, scheme *runtime.Scheme) error {
	fmt.Println("Creating meshsync resource")
	// Set Meshsync instance as the owner and controller
	ctrl.SetControllerReference(m, &appsv1.Deployment{}, scheme)
	return nil
}

// StartDiscovery - run pipelines
func StartDiscovery(config *rest.Config) error {

	// Configure discovery
	client, err := discovery.NewClient(config)
	if err != nil {
		log.Printf("Couldnot create client: %s", err)
		return err
	}

	err = cluster.StartDiscovery(client)
	if err != nil {
		return err
	}

	err = istio.StartDiscovery(client)
	if err != nil {
		return err
	}

	return nil
}

// StartInformer - run informer
func StartInformer(config *rest.Config) error {

	// Configure discovery
	client, err := inf.NewClient(config)
	if err != nil {
		log.Printf("Couldnot create informer client: %s", err)
		return err
	}

	log.Println("start cluster informers")
	cluster.StartInformer(client)

	log.Println("start istio informers")
	istio.StartInformer(client)

	return nil
}
