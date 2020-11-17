package cluster

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type Resources struct {
	Global GlobalResources `json:"global,omitempty"`
	Local  LocalResources  `json:"local,omitempty"`
}

type GlobalResources struct {
	Nodes      []corev1.Node      `json:"nodes,omitempty"`
	Namespaces []corev1.Namespace `json:"namespaces,omitempty"`
}

type LocalResources struct {
	Deployments []appsv1.Deployment `json:"deployments,omitempty"`
	Pods        []corev1.Pod        `json:"pods,omitempty"`
}
