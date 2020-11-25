package meshsync

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
)

func GetResource(m *mesheryv1alpha1.MeshSync) runtime.Object {
	return resource(m.ObjectMeta.Name, m.ObjectMeta.Namespace, m.Spec.Size)
}

// CreateDeployment returns a meshsync Deployment object
func CreateResource(m *mesheryv1alpha1.MeshSync, scheme *runtime.Scheme) *appsv1.Deployment {
	dep := resource(m.ObjectMeta.Name, m.ObjectMeta.Namespace, m.Spec.Size)
	// Set Meshsync instance as the owner and controller
	ctrl.SetControllerReference(m, dep, scheme)
	return dep
}

func resource(name string, namespace string, replicas int32) *appsv1.Deployment {
	labels := map[string]string{
		"app": name,
	}
	image := "layer5io/meshery-meshsync:stable-latest"

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: image,
						Name:  name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 11000,
							Name:          name,
						}},
					}},
				},
			},
		},
	}
	return deployment
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
