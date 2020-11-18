package istio

import (
	"istio.io/client-go/pkg/apis/networking/v1beta1"
)

type Resources struct {
	VirtualServices []v1beta1.VirtualService `json:"virtualservices,omitempty"`
	Sidecars        []v1beta1.Sidecar        `json:"sidecars,omitempty"`
	WorkloadEntries []v1beta1.WorkloadEntry  `json:"workloadentries,omitempty"`
}
