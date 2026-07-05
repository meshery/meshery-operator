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

package meshsync

import (
	"testing"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// testObjName is the name/namespace used for MeshSync CR fixtures in this file.
const testObjName = "test"

func TestGetObjects(t *testing.T) {
	m := &mesheryv1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testObjName,
			Namespace: testObjName,
		},
		Spec: mesheryv1alpha1.MeshSyncSpec{
			Size: 1,
		},
	}
	obj := GetObjects(m, "")
	if len(obj) == 0 {
		t.Fatal("GetObjects returned no objects")
	}
	if obj[0] == nil {
		t.Error("GetObjects returned nil for server object")
	}
}

func TestEnsureNatsScheme(t *testing.T) {
	cases := map[string]string{
		"meshery-nats:4222":  "nats://meshery-nats:4222",
		"10.0.0.1:4222":      "nats://10.0.0.1:4222",
		"nats://broker:4222": "nats://broker:4222",
		"tls://broker:4222":  "tls://broker:4222",
	}
	for in, want := range cases {
		if got := ensureNatsScheme(in); got != want {
			t.Errorf("ensureNatsScheme(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSetBrokerURLByName(t *testing.T) {
	c := &corev1.Container{Env: []corev1.EnvVar{
		{Name: "OTHER", Value: "x"},
		{Name: brokerURLEnv, Value: defaultBrokerURL},
	}}
	setBrokerURL(c, "10.0.0.5:4222", "")
	if c.Env[1].Value != "nats://10.0.0.5:4222" {
		t.Errorf("BROKER_URL = %q, want nats://10.0.0.5:4222", c.Env[1].Value)
	}
	if c.Env[0].Value != "x" {
		t.Errorf("unrelated env var was mutated: %q", c.Env[0].Value)
	}

	// An empty endpoint must leave the template default untouched.
	setBrokerURL(c, "", "")
	if c.Env[1].Value != "nats://10.0.0.5:4222" {
		t.Errorf("empty URL must not change BROKER_URL, got %q", c.Env[1].Value)
	}
}

func TestSetBrokerURLWithTokenSecret(t *testing.T) {
	c := &corev1.Container{Env: []corev1.EnvVar{
		{Name: "OTHER", Value: "x"},
		{Name: brokerURLEnv, Value: defaultBrokerURL},
	}}
	setBrokerURL(c, "meshery-nats.meshery:4222", "meshery-nats-auth")

	var tokenIdx, urlIdx = -1, -1
	for i, e := range c.Env {
		switch e.Name {
		case natsTokenEnv:
			tokenIdx = i
			if e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil ||
				e.ValueFrom.SecretKeyRef.Name != "meshery-nats-auth" ||
				e.ValueFrom.SecretKeyRef.Key != natsTokenKey {
				t.Errorf("NATS_TOKEN must be sourced from the auth Secret, got %+v", e)
			}
		case brokerURLEnv:
			urlIdx = i
			want := "nats://$(NATS_TOKEN)@meshery-nats.meshery:4222"
			if e.Value != want {
				t.Errorf("BROKER_URL = %q, want %q", e.Value, want)
			}
		}
	}
	if tokenIdx == -1 || urlIdx == -1 {
		t.Fatalf("missing NATS_TOKEN (%d) or BROKER_URL (%d) env entry", tokenIdx, urlIdx)
	}
	// $(VAR) references only expand when VAR is defined earlier in the list.
	if tokenIdx > urlIdx {
		t.Errorf("NATS_TOKEN (idx %d) must precede BROKER_URL (idx %d)", tokenIdx, urlIdx)
	}
}

func TestApplyVersion(t *testing.T) {
	c := &corev1.Container{Image: "meshery/meshsync:stable-latest", ImagePullPolicy: corev1.PullAlways}

	applyVersion(c, "")
	if c.Image != "meshery/meshsync:stable-latest" || c.ImagePullPolicy != corev1.PullAlways {
		t.Errorf("empty version must leave the template untouched, got %s/%s", c.Image, c.ImagePullPolicy)
	}

	applyVersion(c, "v1.0.0")
	if c.Image != "meshery/meshsync:v1.0.0" || c.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("pinned version = %s/%s, want meshery/meshsync:v1.0.0/IfNotPresent", c.Image, c.ImagePullPolicy)
	}

	applyVersion(c, "edge-latest")
	if c.Image != "meshery/meshsync:edge-latest" || c.ImagePullPolicy != corev1.PullAlways {
		t.Errorf("moving tag = %s/%s, want meshery/meshsync:edge-latest/Always", c.Image, c.ImagePullPolicy)
	}
}

func TestWithTokenUserinfo(t *testing.T) {
	cases := map[string]string{ //nolint:gosec // G101: fixture URLs with placeholder userinfo, not credentials
		"nats://host:4222":            "nats://$(NATS_TOKEN)@host:4222",
		"tls://host:4222":             "tls://$(NATS_TOKEN)@host:4222",
		"nats://user:pass@host:4222":  "nats://user:pass@host:4222",
		"nats://$(NATS_TOKEN)@h:4222": "nats://$(NATS_TOKEN)@h:4222",
	}
	for in, want := range cases {
		if got := withTokenUserinfo(in); got != want {
			t.Errorf("withTokenUserinfo(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestServesHealthEndpoints locks in the version gate that decides whether a
// MeshSync image is known to serve /healthz and /readyz. Only pinned semantic
// versions at or above minHealthEndpointsVersion (v1.0.1) qualify; moving tags,
// pre-releases of the boundary version, commit shas, and the empty default must
// stay on the exec probe so an httpGet probe never crashloops an image that
// serves nothing on the client port.
func TestServesHealthEndpoints(t *testing.T) {
	cases := map[string]bool{
		"v1.0.1":        true,  // first release with the endpoints
		"1.0.1":         true,  // the leading "v" is optional
		"v1.0.2":        true,  // later patch
		"v1.1.0":        true,  // later minor
		"v2.0.0":        true,  // later major
		"v1.1.0-beta.1": true,  // pre-release that still postdates v1.0.1
		"v1.0.0":        false, // predates the endpoints
		"v0.8.26":       false, // predates the endpoints
		"v1.0.1-rc.1":   false, // pre-release of the boundary version, unprovable
		"stable-latest": false, // moving channel tag, can't be proven
		"edge-latest":   false, // moving channel tag, can't be proven
		"latest":        false, // moving tag
		"":              false, // empty -> template default image
		"abc1234":       false, // commit-sha tag
	}
	for version, want := range cases {
		if got := servesHealthEndpoints(version); got != want {
			t.Errorf("servesHealthEndpoints(%q) = %v, want %v", version, got, want)
		}
	}
}

// TestGetServerObjectProbes verifies the built Deployment carries the right
// probes for the deployed version: httpGet /healthz + /readyz for capable
// versions, and the version-skew-safe exec liveness with no readiness probe for
// everything else.
func TestGetServerObjectProbes(t *testing.T) {
	t.Run("capable version uses httpGet liveness and readiness", func(t *testing.T) {
		c := builtMeshSyncContainer(t, "v1.0.1")
		assertHTTPGetProbe(t, "liveness", c.LivenessProbe, healthzPath)
		assertHTTPGetProbe(t, "readiness", c.ReadinessProbe, readyzPath)
	})

	// Moving tags, pre-endpoint pins, and the empty default must all fall back
	// to the exec liveness probe with no readiness probe.
	for _, version := range []string{"stable-latest", "v1.0.0", ""} {
		version := version
		t.Run("fallback keeps exec liveness and no readiness: "+versionLabel(version), func(t *testing.T) {
			c := builtMeshSyncContainer(t, version)
			if c.LivenessProbe == nil || c.LivenessProbe.Exec == nil {
				t.Fatalf("expected the exec liveness probe, got %+v", c.LivenessProbe)
			}
			if c.LivenessProbe.HTTPGet != nil {
				t.Error("fallback must not switch to an httpGet liveness probe")
			}
			if c.ReadinessProbe != nil {
				t.Errorf("fallback must not attach a readiness probe, got %+v", c.ReadinessProbe)
			}
		})
	}
}

// builtMeshSyncContainer builds the MeshSync Deployment for the given version
// and returns its first (and only) container.
func builtMeshSyncContainer(t *testing.T, version string) corev1.Container {
	t.Helper()
	m := &mesheryv1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{Name: testObjName, Namespace: testObjName},
		Spec:       mesheryv1alpha1.MeshSyncSpec{Size: 1, Version: version},
	}
	dep, ok := GetServerObject(m, "").(*appsv1.Deployment)
	if !ok {
		t.Fatal("GetServerObject did not return a *appsv1.Deployment")
	}
	if len(dep.Spec.Template.Spec.Containers) == 0 {
		t.Fatal("built Deployment has no containers")
	}
	return dep.Spec.Template.Spec.Containers[0]
}

// assertHTTPGetProbe fails unless probe is an httpGet probe hitting path on the
// named client port (and carries no exec handler).
func assertHTTPGetProbe(t *testing.T, kind string, probe *corev1.Probe, path string) {
	t.Helper()
	if probe == nil || probe.HTTPGet == nil {
		t.Fatalf("expected an httpGet %s probe, got %+v", kind, probe)
	}
	if probe.Exec != nil {
		t.Errorf("%s probe must not retain an exec handler", kind)
	}
	if got := probe.HTTPGet.Path; got != path {
		t.Errorf("%s path = %q, want %q", kind, got, path)
	}
	if got := probe.HTTPGet.Port.StrVal; got != clientPortName {
		t.Errorf("%s probe must target the named %q port, got %q", kind, clientPortName, got)
	}
}

// versionLabel renders the empty default version readably in subtest names.
func versionLabel(v string) string {
	if v == "" {
		return "default"
	}
	return v
}
