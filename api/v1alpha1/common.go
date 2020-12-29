package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Healthy    ConditionType = "healthy"
	NotHealthy ConditionType = "not healthy"
	Unknown    ConditionType = "unknown"

	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

type ConditionType string

type ConditionStatus string

type Condition struct {
	Type               ConditionType   `json:"type" yaml:"type"`
	Status             ConditionStatus `json:"status" yaml:"status"`
	ObservedGeneration int64           `json:"observedGeneration,omitempty" yaml:"observedGeneration,omitempty"`
	LastProbeTime      metav1.Time     `json:"lastProbeTime,omitempty" yaml:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time     `json:"lastTransitionTime" yaml:"lastTransitionTime"`
	Reason             string          `json:"reason" yaml:"reason"`
	Message            string          `json:"message" yaml:"message"`
}
