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

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CustomMeshsyncBroker specifies a custom or external NATS broker connection.
type CustomMeshsyncBroker struct {
	// Implementation note (kept out of the generated API docs by the blank line
	// below): the URL is validated at admission so an unrendered placeholder (such
	// as ${PLAYGROUND_NATS_URL}), an incompatible protocol scheme (such as http://),
	// an invalid port, or a path is rejected when the MeshSync is written, instead of
	// trapping the MeshSync pod in CrashLoopBackOff and failing readiness probes.
	// Note: a bare out-of-range dotted quad (e.g. 999.0.0.1:4222) matches the hostname
	// branch because numeric labels are valid in DNS hostnames; bracketed IPv4-embedded
	// forms are strictly validated. The pattern is duplicated in the other API version;
	// TestCustomBrokerURLPattern (api/v1alpha1) keeps the generated copies identical.

	// URL pins the custom NATS broker URL. Must represent a valid NATS endpoint:
	// optionally prefixed with nats:// or tls:// (or a bare host:port), with a valid
	// hostname, FQDN, IPv4, or bracketed IPv6 address and a port between 1 and 65535.
	// +kubebuilder:validation:Pattern=`^((nats|tls)://)?(([a-zA-Z0-9_.~!$&'()*+,;=-]|%[0-9A-Fa-f]{2})+(:([a-zA-Z0-9_.~!$&'()*+,;=-]|%[0-9A-Fa-f]{2})+)?@)?((([a-zA-Z0-9]([-a-zA-Z0-9]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([-a-zA-Z0-9]{0,61}[a-zA-Z0-9])?\.?)|\[(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(:[0-9a-fA-F]{1,4}){1,6}|:((:[0-9a-fA-F]{1,4}){1,7}|:)|(([0-9a-fA-F]{1,4}:){6}|([0-9a-fA-F]{1,4}:){1,5}:|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1}:|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,2}:|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,3}:|[0-9a-fA-F]{1,4}:(:[0-9a-fA-F]{1,4}){1,4}:|::([0-9a-fA-F]{1,4}:){0,5})((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9]))\]):0*([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$`
	// +optional
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
	// Version pins the MeshSync image tag. When empty the operator deploys the
	// pinned default MeshSync release it ships with - never a moving channel
	// tag, so the image a cluster runs cannot change under it on a pod restart.
	// Use the bare semantic version the registry publishes ("1.0.3"); a leading
	// "v" is stripped for you, because meshery/meshsync publishes no
	// v-prefixed image tags even though its releases are named that way. A
	// moving tag ("edge-latest") is honoured if you ask for one explicitly, and
	// is then pulled on every restart.
	// +optional
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
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
// +kubebuilder:storageversion
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
