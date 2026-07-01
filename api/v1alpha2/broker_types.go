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

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BrokerSpec defines the desired state of Broker.
type BrokerSpec struct {
	// Version pins the NATS server image tag. When empty the operator uses its
	// bundled default NATS version.
	// +optional
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Service controls how the broker is exposed on the network. Every field is
	// reconcilable in place: editing it updates the live Service and re-derives
	// status.endpoint without recreating the broker pods.
	// +optional
	Service BrokerServiceSpec `json:"service,omitempty" yaml:"service,omitempty"`

	// Size is the number of NATS server replicas.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=1
	Size int32 `json:"size,omitempty" yaml:"size,omitempty"`
}

// BrokerServiceSpec configures the client-facing NATS Service.
//
// +kubebuilder:validation:XValidation:rule="!has(self.loadBalancerClass) || !has(self.type) || self.type == 'LoadBalancer'",message="loadBalancerClass is only valid when service type is LoadBalancer"
// +kubebuilder:validation:XValidation:rule="!has(self.loadBalancerSourceRanges) || !has(self.type) || self.type == 'LoadBalancer'",message="loadBalancerSourceRanges is only valid when service type is LoadBalancer"
type BrokerServiceSpec struct {
	// Annotations are merged onto the client Service (cloud LB hints, MetalLB
	// address pools, internal-LB switches).
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`

	// LoadBalancerClass selects a specific load-balancer implementation
	// (LoadBalancer type only).
	// +optional
	LoadBalancerClass *string `json:"loadBalancerClass,omitempty" yaml:"loadBalancerClass,omitempty"`

	// Type is the Kubernetes Service type for client access. When empty the
	// operator keeps its historical default (LoadBalancer). Set ClusterIP on
	// clusters without a cloud load-balancer (kind, minikube, bare-metal), or
	// NodePort to expose the broker on node IPs.
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +optional
	Type corev1.ServiceType `json:"type,omitempty" yaml:"type,omitempty"`

	// ExternalEndpointOverride pins the advertised external endpoint (host:port)
	// when auto-derivation is undesirable: an ingress/gateway in front of the
	// broker, an air-gapped topology, or NAT. The nats:// scheme is added by
	// consumers.
	// +optional
	ExternalEndpointOverride string `json:"externalEndpointOverride,omitempty" yaml:"externalEndpointOverride,omitempty"`

	// LoadBalancerSourceRanges restricts client access to the given CIDRs
	// (LoadBalancer type only).
	// +optional
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty" yaml:"loadBalancerSourceRanges,omitempty"`
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
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=br,categories=meshery
// +kubebuilder:printcolumn:name="Size",type=integer,JSONPath=`.spec.size`
// +kubebuilder:printcolumn:name="External",type=string,JSONPath=`.status.endpoint.external`
// +kubebuilder:printcolumn:name="Internal",type=string,JSONPath=`.status.endpoint.internal`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
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
