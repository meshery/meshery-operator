/*
Copyright 2020 Layer5, Inc.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CustomMeshsyncBroker struct {
	URL string `json:"url,omitempty" yaml:"url,omitempty"`
}

type NativeMeshsyncBroker struct {
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

type MeshsyncBroker struct {
	Custom CustomMeshsyncBroker `json:"custom,omitempty" yaml:"custom,omitempty"`
	Native NativeMeshsyncBroker `json:"native,omitempty" yaml:"native,omitempty"`
}

// MeshSyncSpec defines the desired state of MeshSync
type MeshSyncSpec struct {
	WatchList corev1.ConfigMap `json:"watch-list,omitempty" yaml:"watch-list,omitempty"`
	Broker    MeshsyncBroker   `json:"broker,omitempty" yaml:"broker,omitempty"`
	Version   string           `json:"version,omitempty" yaml:"version,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	Size int32 `json:"size,omitempty" yaml:"size,omitempty"`
}

// MeshSyncStatus defines the observed state of MeshSync
type MeshSyncStatus struct {
	PublishingTo string             `json:"publishing-to,omitempty" yaml:"publishing-to,omitempty"`
	Conditions   []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// MeshSync is the Schema for the meshsyncs API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type MeshSync struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   MeshSyncSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status MeshSyncStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// MeshSyncList contains a list of MeshSync
// +kubebuilder:object:root=true
type MeshSyncList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items           []MeshSync `json:"items" yaml:"items"`
}

func init() {
	SchemeBuilder.Register(&MeshSync{}, &MeshSyncList{})
}
