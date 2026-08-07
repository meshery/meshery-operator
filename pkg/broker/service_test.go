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
	"strings"
	"testing"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	"github.com/meshery/meshery-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestApplyServiceSpec(t *testing.T) {
	lb := corev1.ServiceTypeLoadBalancer
	class := "internal"

	t.Run("unset type keeps the chart default (ClusterIP)", func(t *testing.T) {
		svc := &corev1.Service{Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP}}
		applyServiceSpec(svc, mesheryv1alpha1.BrokerServiceSpec{})
		if svc.Spec.Type != corev1.ServiceTypeClusterIP {
			t.Errorf("type = %q, want ClusterIP (no implicit LoadBalancer)", svc.Spec.Type)
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

// TestBrokerImagePinning covers the broker half of the managed-component image
// contract: the chart-rendered default must be an immutable tag (never a
// moving channel tag that would re-point under a running cluster), spec.version
// must be honoured, and the pull policy must follow the tag's mutability.
func TestBrokerImagePinning(t *testing.T) {
	t.Run("chart default is a pinned tag", func(t *testing.T) {
		c := natsContainer(t, brokerFixture(1, corev1.ServiceTypeClusterIP))
		tag, ok := imageTag(c.Image)
		if !ok {
			t.Fatalf("chart-rendered image %q carries no tag", c.Image)
		}
		if utils.MovingTag(tag) {
			t.Errorf("chart-rendered image %q uses a moving tag; bump NATS_CHART_VERSION and re-run make nats-manifests instead", c.Image)
		}
		if c.ImagePullPolicy == corev1.PullAlways {
			t.Errorf("pinned default must not carry imagePullPolicy: Always (got %q)", c.ImagePullPolicy)
		}
	})

	cases := []struct {
		version    string
		wantImage  string
		wantPolicy corev1.PullPolicy
	}{
		{"2.10.22-alpine", "nats:2.10.22-alpine", corev1.PullIfNotPresent},
		// Registry convention: library/nats publishes no v-prefixed tags, so a
		// v-prefixed pin is normalised rather than left to ImagePullBackOff.
		{"v2.10.22", "nats:2.10.22", corev1.PullIfNotPresent},
		{"latest", "nats:latest", corev1.PullAlways},
		{"edge-latest", "nats:edge-latest", corev1.PullAlways},
	}
	for _, tc := range cases {
		t.Run("spec.version "+tc.version, func(t *testing.T) {
			m := brokerFixture(1, corev1.ServiceTypeClusterIP)
			m.Spec.Version = tc.version
			c := natsContainer(t, m)
			if c.Image != tc.wantImage {
				t.Errorf("image = %q, want %q", c.Image, tc.wantImage)
			}
			if c.ImagePullPolicy != tc.wantPolicy {
				t.Errorf("imagePullPolicy = %q, want %q", c.ImagePullPolicy, tc.wantPolicy)
			}
		})
	}
}

// natsContainer builds the Broker's objects and returns the NATS server
// container from the rendered StatefulSet.
func natsContainer(t *testing.T, m *mesheryv1alpha1.Broker) corev1.Container {
	t.Helper()
	sts, _ := findStatefulSetAndClientService(GetObjects(m))
	if sts == nil {
		t.Fatalf("no StatefulSet named %q among chart objects", natsServiceName)
	}
	for _, c := range sts.Spec.Template.Spec.Containers {
		if c.Name == natsName {
			return c
		}
	}
	t.Fatalf("no %q container in the rendered StatefulSet", natsName)
	return corev1.Container{}
}

// imageTag splits the tag off a "repo:tag" reference (the managed images carry
// no registry host, so the last colon is unambiguous).
func imageTag(image string) (string, bool) {
	i := strings.LastIndex(image, ":")
	if i < 0 {
		return "", false
	}
	return image[i+1:], true
}

func brokerFixture(size int32, svcType corev1.ServiceType) *mesheryv1alpha1.Broker {
	m := &mesheryv1alpha1.Broker{}
	m.Name = testBrokerName
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
	// tokenPrefix (see broker.go) + 32 random bytes, hex-encoded. Derived from
	// tokenPrefix so it tracks the constant instead of drifting when the prefix
	// changes (it was added after this test and silently broke the old literal 64).
	const wantLen = len(tokenPrefix) + 64
	if len(a) != wantLen {
		t.Errorf("token length = %d, want %d", len(a), wantLen)
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
