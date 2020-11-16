package meshsync

import (
	"fmt"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
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
