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
	"fmt"
	"net"
	"net/url"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

// DeriveEndpoint computes the broker's internal and external host:port endpoints
// directly from the Service object, with no network I/O. It is a pure function,
// so it is safe to call on the reconcile path (no blocking TCP dials, which
// previously came from meshkit's GetEndpoint) and is trivially unit-testable
// across Service types (WS-3 §6.2.3, WS-6).
//
// Returned addresses are host:port (no scheme); callers add the nats:// scheme
// at the point of use. external is empty when the Service exposes no external
// address yet. pending is true specifically when a LoadBalancer Service has not
// been assigned ingress yet, signalling the caller that the address will arrive
// via a later Service watch event rather than needing a busy requeue.
func DeriveEndpoint(svc *corev1.Service, apiServerURL string) (internal, external string, pending bool) {
	port := clientServicePort(svc)
	if port == nil {
		return "", "", false
	}

	internal = net.JoinHostPort(internalHost(svc), strconv.Itoa(int(port.Port)))

	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		if host := loadBalancerHost(svc); host != "" {
			external = net.JoinHostPort(host, strconv.Itoa(int(port.Port)))
		} else {
			pending = true
		}
	case corev1.ServiceTypeNodePort:
		if port.NodePort > 0 {
			if host := apiServerHostOnly(apiServerURL); host != "" {
				external = net.JoinHostPort(host, strconv.Itoa(int(port.NodePort)))
			}
		}
	}

	return internal, external, pending
}

// clientServicePort returns the NATS client port, preferring the port named
// "client" and falling back to the first declared port.
func clientServicePort(svc *corev1.Service) *corev1.ServicePort {
	if svc == nil || len(svc.Spec.Ports) == 0 {
		return nil
	}
	for i := range svc.Spec.Ports {
		if svc.Spec.Ports[i].Name == clientPortName {
			return &svc.Spec.Ports[i]
		}
	}
	return &svc.Spec.Ports[0]
}

// internalHost returns the in-cluster address for the Service: the ClusterIP
// when assigned, otherwise the stable cluster DNS name (headless or not-yet-
// allocated Services).
func internalHost(svc *corev1.Service) string {
	if ip := svc.Spec.ClusterIP; ip != "" && ip != corev1.ClusterIPNone {
		return ip
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace)
}

// loadBalancerHost returns the first assigned LoadBalancer ingress address
// (IP preferred, hostname otherwise), or "" while the address is pending.
func loadBalancerHost(svc *corev1.Service) string {
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			return ing.IP
		}
		if ing.Hostname != "" {
			return ing.Hostname
		}
	}
	return ""
}

// apiServerHostOnly extracts the bare host from the API server URL, used as the
// node address for NodePort Services (clusters without a cloud LoadBalancer).
func apiServerHostOnly(apiServerURL string) string {
	if apiServerURL == "" {
		return ""
	}
	if u, err := url.Parse(apiServerURL); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	if host, _, err := net.SplitHostPort(apiServerURL); err == nil {
		return host
	}
	return apiServerURL
}
