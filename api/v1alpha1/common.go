package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionTrue    metav1.ConditionStatus = "True"
	ConditionFalse   metav1.ConditionStatus = "False"
	ConditionUnknown metav1.ConditionStatus = "Unknown"
)
