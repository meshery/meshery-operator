package meshsync

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	val1     int32 = 1
	val60    int64 = 60
	val11000 int32 = 11000

	valtrue bool = true

	MesheryLabel = map[string]string{
		"app": "meshery",
	}

	MeshSyncLabel = map[string]string{
		"app":       MesheryLabel["app"],
		"component": "meshsync",
	}

	Deployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "meshery-meshsync",
			Labels: MeshSyncLabel,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &val1,
			Selector: &metav1.LabelSelector{
				MatchLabels: MeshSyncLabel,
			},
			Template: PodTemplate,
		},
	}

	PodTemplate = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "meshery-meshsync",
			Labels: MeshSyncLabel,
		},
		Spec: corev1.PodSpec{
			ShareProcessNamespace:         &valtrue,
			TerminationGracePeriodSeconds: &val60,
			Containers: []corev1.Container{
				{
					Name:            "meshsync",
					Image:           "layer5/meshery-meshsync:stable-latest",
					ImagePullPolicy: corev1.PullAlways,
					Ports: []corev1.ContainerPort{
						{
							Name:          "client",
							HostPort:      val11000,
							ContainerPort: val11000,
						},
					},
					Command: []string{
						"./meshery-meshsync", "--broker-url", "$(BROKER_URL)",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "BROKER_URL",
							Value: "http://localhost:4222",
						},
					},
				},
			},
		},
	}
)
