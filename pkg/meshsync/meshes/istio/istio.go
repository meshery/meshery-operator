package istio

import (
	networkingV1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	networkingV1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	securityV1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
)

type Resources struct {
	VirtualServices        []networkingV1beta1.VirtualService      `json:"virtualservices,omitempty"`
	Sidecars               []networkingV1beta1.Sidecar             `json:"sidecars,omitempty"`
	WorkloadEntries        []networkingV1beta1.WorkloadEntry       `json:"workloadentries,omitempty"`
	AuthorizationPolicies  []securityV1beta1.AuthorizationPolicy   `json:"authorizationpolicies,omitempty"`
	DestinationRules       []networkingV1beta1.DestinationRule     `json:"destinationrules,omitempty"`
	EnvoyFilters           []networkingV1alpha3.EnvoyFilter        `json:"envoyfilters,omitempty"`
	Gateways               []networkingV1beta1.Gateway             `json:"gateways,omitempty"`
	PeerAuthentications    []securityV1beta1.PeerAuthentication    `json:"peerauthenticatons,omitempty"`
	RequestAuthentications []securityV1beta1.RequestAuthentication `json:"requestauthentications,omitempty"`
	ServiceEntries         []networkingV1beta1.ServiceEntry        `json:"serviceentries,omitempty"`
	WorkloadGroups         []networkingV1alpha3.WorkloadGroup      `json:"workloadgroups,omitempty"`
}
