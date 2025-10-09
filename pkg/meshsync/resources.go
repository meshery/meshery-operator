/*
Copyright Meshery Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package meshsync

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	MesheryAnnotation = map[string]string{
		"meshery/component-type": "management-plane",
	}

	// Resource limits and requests
	CPURequest    = resource.MustParse("500m")
	CPULimit      = resource.MustParse("4")
	MemoryRequest = resource.MustParse("512Mi")
	MemoryLimit   = resource.MustParse("4Gi")

	Deployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "meshery-meshsync",
			Labels:      MeshSyncLabel,
			Annotations: MesheryAnnotation,
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
			Name:        "meshery-meshsync",
			Labels:      MeshSyncLabel,
			Annotations: MesheryAnnotation,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName:            "meshery-operator",
			ShareProcessNamespace:         &valtrue,
			TerminationGracePeriodSeconds: &val60,
			Containers: []corev1.Container{
				{
					Name:            "meshsync",
					Image:           "meshery/meshsync:stable-latest",
					ImagePullPolicy: corev1.PullAlways,
					Ports: []corev1.ContainerPort{
						{
							Name:          "client",
							HostPort:      val11000,
							ContainerPort: val11000,
						},
					},
					Command: []string{
						"./meshery-meshsync",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "BROKER_URL",
							Value: "http://localhost:4222",
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    CPURequest,
							corev1.ResourceMemory: MemoryRequest,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    CPULimit,
							corev1.ResourceMemory: MemoryLimit,
						},
					},
					LivenessProbe: &corev1.Probe{
						InitialDelaySeconds: 60,
						PeriodSeconds:       10,
						TimeoutSeconds:      2,
						FailureThreshold:    4,
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"./meshery-meshsync",
									"-h",
								},
							},
						},
					},
					ReadinessProbe: &corev1.Probe{
						InitialDelaySeconds: 20,
						PeriodSeconds:       4,
						TimeoutSeconds:      2,
						FailureThreshold:    4,
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"./meshery-meshsync",
									"-h",
								},
							},
						},
					},
				},
			},
		},
	}
)
