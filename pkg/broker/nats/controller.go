package nats

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
)

func GetResource(m *mesheryv1alpha1.Broker) runtime.Object {
	return resource(m.ObjectMeta.Name, m.ObjectMeta.Namespace, m.Spec.Size)
}

// CreateDeployment returns a meshsync Deployment object
func CreateResource(m *mesheryv1alpha1.Broker, scheme *runtime.Scheme) *appsv1.Deployment {
	dep := resource(m.ObjectMeta.Name, m.ObjectMeta.Namespace, m.Spec.Size)
	// Set Meshsync instance as the owner and controller
	_ = ctrl.SetControllerReference(m, dep, scheme)
	return dep
}

func resource(name, namespace string, replicas int32) *appsv1.Deployment {
	labels := map[string]string{
		"app": name,
	}
	image := "nats:2.1-alpine"

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
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 4222,
								Name:          "client",
							},
							{
								ContainerPort: 8222,
								Name:          "http",
							},
							{
								ContainerPort: 6222,
								Name:          "routing",
							},
						},
					}},
				},
			},
		},
	}
	return deployment
}
