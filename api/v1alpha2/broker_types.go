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
	// Version pins the NATS server image tag. When empty the operator uses the
	// pinned NATS version from its vendored chart - never a moving channel tag.
	// Use the bare tag the registry publishes ("2.14.2-alpine"); a leading "v"
	// is stripped for you, because library/nats publishes no v-prefixed image
	// tags. A moving tag ("latest") is honoured if you ask for one explicitly,
	// and is then pulled on every restart.
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
	// broker stays cluster-internal (ClusterIP). Set LoadBalancer to acquire a
	// cloud load-balancer address, or NodePort to expose the broker on node
	// IPs.
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +optional
	Type corev1.ServiceType `json:"type,omitempty" yaml:"type,omitempty"`

	// ExternalEndpointOverride pins the advertised external endpoint (host:port)
	// when auto-derivation is undesirable: an ingress/gateway in front of the
	// broker, an air-gapped topology, or NAT. The nats:// scheme is added by
	// consumers.
	// +optional
	ExternalEndpointOverride string `json:"externalEndpointOverride,omitempty" yaml:"externalEndpointOverride,omitempty"`

	// Implementation note (kept out of the generated API docs by the blank line
	// below): entries are validated at admission so a malformed value - an
	// unrendered template placeholder, a bare IP, a hostname - is rejected when
	// the Broker is written, instead of surfacing as a perpetual reconcile error
	// once the operator applies the client Service. Kubernetes 1.31+ has CEL
	// isCIDR(), but older API servers reject any CRD that uses it, so this is an
	// OpenAPI pattern, which every Kubernetes version accepts. The pattern is
	// duplicated in the other API version; TestLoadBalancerSourceRangesPattern
	// (api/v1alpha1) keeps the generated copies identical and checks them
	// against net.ParseCIDR.

	// LoadBalancerSourceRanges restricts client access to the given CIDRs
	// (LoadBalancer type only). Each entry must be an IPv4 or IPv6 CIDR, such as
	// 203.0.113.0/24 or 2001:db8::/32.
	// +kubebuilder:validation:items:Pattern=`^(((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])/(3[0-2]|[12][0-9]|[0-9])|(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(:[0-9a-fA-F]{1,4}){1,6}|:((:[0-9a-fA-F]{1,4}){1,7}|:)|(([0-9a-fA-F]{1,4}:){6}|([0-9a-fA-F]{1,4}:){1,5}:|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1}:|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,2}:|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,3}:|[0-9a-fA-F]{1,4}:(:[0-9a-fA-F]{1,4}){1,4}:|::([0-9a-fA-F]{1,4}:){0,5})((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9]))/(12[0-8]|1[01][0-9]|[1-9][0-9]|[0-9]))$`
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
