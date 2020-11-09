package controllers

import (
	"fmt"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	// ctrl "sigs.k8s.io/controller-runtime"
	appsv1 "k8s.io/api/apps/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// deploymentForMemcached returns a memcached Deployment object
func createDeployment(m *mesheryv1alpha1.MeshSync, scheme *runtime.Scheme) *appsv1.Deployment {
	fmt.Println("Creating meshsync")
	// ls := labels(m.Name)
	// replicas := m.Spec.Size

	// dep := &appsv1.Deployment{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      m.Name,
	// 		Namespace: m.Namespace,
	// 	},
	// 	Spec: appsv1.DeploymentSpec{
	// 		Replicas: &replicas,
	// 		Selector: &metav1.LabelSelector{
	// 			MatchLabels: ls,
	// 		},
	// 		Template: corev1.PodTemplateSpec{
	// 			ObjectMeta: metav1.ObjectMeta{
	// 				Labels: ls,
	// 			},
	// 			Spec: corev1.PodSpec{
	// 				Containers: []corev1.Container{{
	// 					Image:   "meshsync:stable-latest",
	// 					Name:    "meshsync",
	// 					Command: []string{"memcached", "-m=64", "-o", "modern", "-v"},
	// 					Ports: []corev1.ContainerPort{{
	// 						ContainerPort: 11211,
	// 						Name:          "memcached",
	// 					}},
	// 				}},
	// 			},
	// 		},
	// 	},
	// }
	// // Set Memcached instance as the owner and controller
	// ctrl.SetControllerReference(m, dep, scheme)
	// return dep
	return nil
}
