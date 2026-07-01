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

package broker

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestDeriveEndpoint exercises the pure, non-blocking endpoint derivation across
// every Service type. It performs no network or apiserver I/O.
func TestDeriveEndpoint(t *testing.T) {
	const apiServerURL = "https://10.20.30.40:6443"
	const clusterIPInternal = "10.96.0.10:4222"

	svc := func(typ corev1.ServiceType, clusterIP string, ports []corev1.ServicePort, ingress ...corev1.LoadBalancerIngress) *corev1.Service {
		s := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "meshery-nats", Namespace: "meshery"},
			Spec:       corev1.ServiceSpec{Type: typ, ClusterIP: clusterIP, Ports: ports},
		}
		s.Status.LoadBalancer.Ingress = ingress
		return s
	}
	clientPort := []corev1.ServicePort{
		{Name: clientPortName, Port: 4222, NodePort: 30422},
		{Name: monitorPortName, Port: 8222},
	}

	cases := []struct {
		name         string
		svc          *corev1.Service
		wantInternal string
		wantExternal string
		wantPending  bool
	}{
		{
			name:         "ClusterIP exposes internal only",
			svc:          svc(corev1.ServiceTypeClusterIP, "10.96.0.10", clientPort),
			wantInternal: clusterIPInternal,
			wantExternal: "",
		},
		{
			name:         "NodePort uses the API server host and node port",
			svc:          svc(corev1.ServiceTypeNodePort, "10.96.0.10", clientPort),
			wantInternal: clusterIPInternal,
			wantExternal: "10.20.30.40:30422",
		},
		{
			name:         "LoadBalancer with an ingress IP",
			svc:          svc(corev1.ServiceTypeLoadBalancer, "10.96.0.10", clientPort, corev1.LoadBalancerIngress{IP: "203.0.113.7"}),
			wantInternal: clusterIPInternal,
			wantExternal: "203.0.113.7:4222",
		},
		{
			name:         "LoadBalancer with an ingress hostname",
			svc:          svc(corev1.ServiceTypeLoadBalancer, "10.96.0.10", clientPort, corev1.LoadBalancerIngress{Hostname: "nats.lb.example.com"}),
			wantInternal: clusterIPInternal,
			wantExternal: "nats.lb.example.com:4222",
		},
		{
			name:         "LoadBalancer pending ingress",
			svc:          svc(corev1.ServiceTypeLoadBalancer, "10.96.0.10", clientPort),
			wantInternal: clusterIPInternal,
			wantExternal: "",
			wantPending:  true,
		},
		{
			name:         "headless service falls back to cluster DNS",
			svc:          svc(corev1.ServiceTypeClusterIP, corev1.ClusterIPNone, clientPort),
			wantInternal: "meshery-nats.meshery.svc.cluster.local:4222",
			wantExternal: "",
		},
		{
			name:         "missing client port falls back to the first port",
			svc:          svc(corev1.ServiceTypeClusterIP, "10.96.0.10", []corev1.ServicePort{{Name: monitorPortName, Port: 8222}}),
			wantInternal: "10.96.0.10:8222",
			wantExternal: "",
		},
		{
			name:         "no ports yields empty endpoints",
			svc:          svc(corev1.ServiceTypeClusterIP, "10.96.0.10", nil),
			wantInternal: "",
			wantExternal: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			internal, external, pending := DeriveEndpoint(tc.svc, apiServerURL)
			if internal != tc.wantInternal {
				t.Errorf("internal = %q, want %q", internal, tc.wantInternal)
			}
			if external != tc.wantExternal {
				t.Errorf("external = %q, want %q", external, tc.wantExternal)
			}
			if pending != tc.wantPending {
				t.Errorf("pending = %v, want %v", pending, tc.wantPending)
			}
		})
	}
}
