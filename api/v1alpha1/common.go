package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Condition struct {
	Type               string                 `json:"type"`
	Status             metav1.ConditionStatus `json:"status"`
	ObservedGeneration int64                  `json:"observedGeneration,omitempty"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime"`
	Reason             string                 `json:"reason"`
	Message            string                 `json:"message"`
}
