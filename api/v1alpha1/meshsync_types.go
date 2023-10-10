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

// an array of resources that meshsync listes to and publishes their events
type PipelineConfigs []PipelineConfig

// resources that meshsync observes and publishes to a given subscriber via the broker
type PipelineConfig struct {
	Name      string `json:"name" yaml:"name"`
	PublishTo string `json:"publish-to" yaml:"publish-to"`
}

// an array of resources that meshsync listens to
type ListenerConfigs []ListenerConfig

// configures resources the meshsync subsribes to
type ListenerConfig struct {
	Name           string `json:"name" yaml:"name"`
	ConnectionName string `json:"connection-name" yaml:"connection-name"`
	PublishTo      string `json:"publish-to" yaml:"publish-to"`
	SubscribeTo    string `json:"subscribe-to" yaml:"subscribe-to"`
}

// Meshsync configuration controls the resources meshsync produces and consumes
type MeshsyncConfig struct {
	PipelineConfigs map[string]PipelineConfigs `json:"pipeline-configs,omitempty" yaml:"pipeline-configs,omitempty"`
	ListenerConfigs map[string]ListenerConfig  `json:"listener-config,omitempty" yaml:"listener-config,omitempty"`
}

// MeshSyncSpec defines the desired state of MeshSync
type MeshSyncSpec struct {
	Size   int32          `json:"size,omitempty" yaml:"size,omitempty"`
	Broker MeshsyncBroker `json:"broker,omitempty" yaml:"broker,omitempty"`
	Config MeshsyncConfig `json:"config,omitempty" yaml:"config,omitempty"`
}

// MeshSyncStatus defines the observed state of MeshSync
type MeshSyncStatus struct {
	PublishingTo string      `json:"publishing-to,omitempty" yaml:"publishing-to,omitempty"`
	Conditions   []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
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
