/*
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

// BrokerSpec defines the desired state of Broker
type BrokerSpec struct {
	Size int32 `json:"size,omitempty" yaml:"size,omitempty"`
}

type Endpoint struct {
	Internal string `json:"internal,omitempty" yaml:"internal,omitempty"`
	External string `json:"external,omitempty" yaml:"external,omitempty"`
}

// BrokerStatus defines the observed state of Broker
type BrokerStatus struct {
	Endpoint   Endpoint           `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// Broker is the Schema for the brokers API
// +kubebuilder:subresource:status
// +kubebuilder:object:root=true
type Broker struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Status            BrokerStatus `json:"status,omitempty" yaml:"status,omitempty"`
	Spec              BrokerSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// BrokerList contains a list of Broker
type BrokerList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items           []Broker `json:"items" yaml:"items"`
}

func init() {
	SchemeBuilder.Register(&Broker{}, &BrokerList{})
}
