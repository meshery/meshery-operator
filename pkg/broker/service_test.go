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

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestNATSImage(t *testing.T) {
	if got, want := natsImage(""), "nats:"+defaultNATSVersion; got != want {
		t.Errorf("natsImage(\"\") = %q, want %q", got, want)
	}
	if got, want := natsImage("2.11.0"), "nats:2.11.0"; got != want {
		t.Errorf("natsImage(version) = %q, want %q", got, want)
	}
}

func TestApplyServiceSpec(t *testing.T) {
	lb := corev1.ServiceTypeLoadBalancer
	class := "internal"

	t.Run("unset type preserves the LoadBalancer default", func(t *testing.T) {
		svc := &corev1.Service{Spec: corev1.ServiceSpec{Type: lb}}
		applyServiceSpec(svc, mesheryv1alpha1.BrokerServiceSpec{})
		if svc.Spec.Type != lb {
			t.Errorf("type = %q, want LoadBalancer", svc.Spec.Type)
		}
	})

	t.Run("explicit ClusterIP overrides and ignores LB-only fields", func(t *testing.T) {
		svc := &corev1.Service{Spec: corev1.ServiceSpec{Type: lb}}
		applyServiceSpec(svc, mesheryv1alpha1.BrokerServiceSpec{
			Type:              corev1.ServiceTypeClusterIP,
			LoadBalancerClass: &class,
			Annotations:       map[string]string{"a": "b"},
		})
		if svc.Spec.Type != corev1.ServiceTypeClusterIP {
			t.Errorf("type = %q, want ClusterIP", svc.Spec.Type)
		}
		if svc.Spec.LoadBalancerClass != nil {
			t.Errorf("loadBalancerClass must not be applied for ClusterIP")
		}
		if svc.Annotations["a"] != "b" {
			t.Errorf("annotation a = %q, want b", svc.Annotations["a"])
		}
	})

	t.Run("LoadBalancer applies class and source ranges", func(t *testing.T) {
		svc := &corev1.Service{Spec: corev1.ServiceSpec{Type: lb}}
		applyServiceSpec(svc, mesheryv1alpha1.BrokerServiceSpec{
			Type:                     lb,
			LoadBalancerClass:        &class,
			LoadBalancerSourceRanges: []string{"10.0.0.0/8"},
		})
		if svc.Spec.LoadBalancerClass == nil || *svc.Spec.LoadBalancerClass != class {
			t.Errorf("loadBalancerClass = %v, want %q", svc.Spec.LoadBalancerClass, class)
		}
		if len(svc.Spec.LoadBalancerSourceRanges) != 1 {
			t.Errorf("sourceRanges = %v, want one entry", svc.Spec.LoadBalancerSourceRanges)
		}
	})
}
