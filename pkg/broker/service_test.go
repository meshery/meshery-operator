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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestApplyServiceSpec(t *testing.T) {
	lb := corev1.ServiceTypeLoadBalancer
	class := "internal"

	t.Run("unset type defaults to LoadBalancer", func(t *testing.T) {
		svc := &corev1.Service{Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP}}
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

// TestGetObjectsFromChart verifies the embedded chart decodes and that the
// BrokerSpec overlay is applied to the right objects.
func TestGetObjectsFromChart(t *testing.T) {
	m := brokerFixture(3, corev1.ServiceTypeClusterIP)
	objs := GetObjects(m)
	if len(objs) < 3 {
		t.Fatalf("expected the vendored chart objects, got %d", len(objs))
	}
	for _, o := range objs {
		if o.GetNamespace() != mesheryName {
			t.Errorf("object %q namespace = %q, want %q", o.GetName(), o.GetNamespace(), mesheryName)
		}
	}

	sts, clientSvc := findStatefulSetAndClientService(objs)
	if sts == nil {
		t.Fatalf("no StatefulSet named %q among chart objects", natsServiceName)
	}
	if sts.Spec.Replicas == nil || *sts.Spec.Replicas != 3 {
		t.Errorf("StatefulSet replicas = %v, want 3 (from spec.size)", sts.Spec.Replicas)
	}
	if clientSvc == nil {
		t.Fatalf("no client Service named %q among chart objects", natsServiceName)
	}
	if clientSvc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("client Service type = %q, want ClusterIP (from spec)", clientSvc.Spec.Type)
	}
}

// TestChartNatsReadsTokenFromSecret asserts the vendored server reads its token
// from a Secret and carries no committed credentials.
func TestChartNatsReadsTokenFromSecret(t *testing.T) {
	sts, _ := findStatefulSetAndClientService(GetObjects(brokerFixture(1, corev1.ServiceTypeClusterIP)))
	if sts == nil {
		t.Fatal("no StatefulSet among chart objects")
	}
	if !natsContainerReadsToken(sts) {
		t.Errorf("nats container should read NATS_TOKEN from a Secret (no committed credentials)")
	}
}

func brokerFixture(size int32, svcType corev1.ServiceType) *mesheryv1alpha1.Broker {
	m := &mesheryv1alpha1.Broker{}
	m.Name = "meshery-broker"
	m.Namespace = mesheryName
	m.Spec = mesheryv1alpha1.BrokerSpec{Size: size, Service: mesheryv1alpha1.BrokerServiceSpec{Type: svcType}}
	return m
}

func findStatefulSetAndClientService(objs []Object) (*appsv1.StatefulSet, *corev1.Service) {
	var sts *appsv1.StatefulSet
	var svc *corev1.Service
	for _, o := range objs {
		switch x := o.(type) {
		case *appsv1.StatefulSet:
			if x.Name == natsServiceName {
				sts = x
			}
		case *corev1.Service:
			if x.Name == natsServiceName {
				svc = x
			}
		}
	}
	return sts, svc
}

func natsContainerReadsToken(sts *appsv1.StatefulSet) bool {
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name != natsName {
			continue
		}
		for _, e := range c.Env {
			if e.Name == "NATS_TOKEN" && e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
				return true
			}
		}
	}
	return false
}

func TestGenerateToken(t *testing.T) {
	a, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 64 { // 32 random bytes hex-encoded
		t.Errorf("token length = %d, want 64", len(a))
	}
	b, _ := GenerateToken()
	if a == b {
		t.Errorf("GenerateToken returned identical tokens")
	}
}

func TestBuildAuthSecret(t *testing.T) {
	s := BuildAuthSecret("meshery", "deadbeef")
	if s.Name != AuthSecretName || s.Namespace != "meshery" {
		t.Errorf("unexpected secret name/namespace: %s/%s", s.Namespace, s.Name)
	}
	if s.StringData["token"] != "deadbeef" {
		t.Errorf("token = %q, want deadbeef", s.StringData["token"])
	}
}
