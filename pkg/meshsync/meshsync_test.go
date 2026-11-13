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
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	"github.com/meshery/meshery-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// testObjName is the name/namespace used for MeshSync CR fixtures in this file.
const testObjName = "test"

// Representative MeshSync image tags: two moving channel tags, an immutable
// release in both the registry's bare form and the v-prefixed form users copy
// off a GitHub release, and a commit-sha tag.
const (
	tagStableLatest = "stable-latest"
	tagEdgeLatest   = "edge-latest"
	tagPinned       = "1.0.2"
	tagPinnedV      = "v1.0.2"
	tagSha          = "abc1234"
)

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

	applyVersion(c, "1.0.0")
	if c.Image != "meshery/meshsync:1.0.0" || c.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("pinned version = %s/%s, want meshery/meshsync:1.0.0/IfNotPresent", c.Image, c.ImagePullPolicy)
	}

	applyVersion(c, tagEdgeLatest)
	if c.Image != "meshery/meshsync:edge-latest" || c.ImagePullPolicy != corev1.PullAlways {
		t.Errorf("moving tag = %s/%s, want meshery/meshsync:edge-latest/Always", c.Image, c.ImagePullPolicy)
	}
}

func TestApplyImage(t *testing.T) {
	c := &corev1.Container{Image: "meshery/meshsync:stable-latest", ImagePullPolicy: corev1.PullAlways}

	applyImage(c, "", "")
	if c.Image != "meshery/meshsync:stable-latest" || c.ImagePullPolicy != corev1.PullAlways {
		t.Errorf("empty repo and version must leave template untouched, got %s/%s", c.Image, c.ImagePullPolicy)
	}

	applyImage(c, "myregistry.corp.internal/meshery/meshsync", "")
	if c.Image != "myregistry.corp.internal/meshery/meshsync:"+defaultMeshSyncVersion || c.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("custom repo with default tag = %s/%s, want myregistry.corp.internal/meshery/meshsync:%s/IfNotPresent",
			c.Image, c.ImagePullPolicy, defaultMeshSyncVersion)
	}

	applyImage(c, "myregistry.corp.internal/meshery/meshsync", "1.0.4")
	if c.Image != "myregistry.corp.internal/meshery/meshsync:1.0.4" || c.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("custom repo with pinned tag = %s/%s, want myregistry.corp.internal/meshery/meshsync:1.0.4/IfNotPresent",
			c.Image, c.ImagePullPolicy)
	}

	applyImage(c, "registry.corp.internal:5000/meshery/meshsync", "v1.0.5")
	if c.Image != "registry.corp.internal:5000/meshery/meshsync:1.0.5" || c.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Errorf("custom repo with port = %s/%s, want registry.corp.internal:5000/meshery/meshsync:1.0.5/IfNotPresent",
			c.Image, c.ImagePullPolicy)
	}
}

func TestGetServerObjectRepository(t *testing.T) {
	t.Run("custom repo with explicit version", func(t *testing.T) {
		m := &mesheryv1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{Name: "meshery-meshsync", Namespace: "meshery"},
			Spec: mesheryv1alpha1.MeshSyncSpec{
				Repository: "myregistry.corp.internal/custom-sync",
				Version:    "1.0.4",
			},
		}
		obj := GetServerObject(m, "")
		deploy := obj.(*appsv1.Deployment)
		c := deploy.Spec.Template.Spec.Containers[0]
		if want := "myregistry.corp.internal/custom-sync:1.0.4"; c.Image != want {
			t.Errorf("image = %q, want %q", c.Image, want)
		}
		if c.ImagePullPolicy != corev1.PullIfNotPresent {
			t.Errorf("pullPolicy = %v, want IfNotPresent", c.ImagePullPolicy)
		}
		if c.ReadinessProbe == nil || c.LivenessProbe == nil || c.LivenessProbe.HTTPGet == nil {
			t.Error("expected HTTP health probes for version 1.0.4")
		}
	})

	t.Run("full-image repository override preserves embedded tag when spec.version is empty", func(t *testing.T) {
		m := &mesheryv1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{Name: "meshery-meshsync", Namespace: "meshery"},
			Spec: mesheryv1alpha1.MeshSyncSpec{
				Repository: "myregistry.corp.internal/custom-sync:1.0.4",
			},
		}
		obj := GetServerObject(m, "")
		deploy := obj.(*appsv1.Deployment)
		c := deploy.Spec.Template.Spec.Containers[0]
		if want := "myregistry.corp.internal/custom-sync:1.0.4"; c.Image != want {
			t.Errorf("image = %q, want %q", c.Image, want)
		}
		if c.ImagePullPolicy != corev1.PullIfNotPresent {
			t.Errorf("pullPolicy = %v, want IfNotPresent", c.ImagePullPolicy)
		}
		if c.ReadinessProbe == nil || c.LivenessProbe == nil || c.LivenessProbe.HTTPGet == nil {
			t.Error("expected HTTP health probes derived from embedded tag 1.0.4")
		}
	})

	t.Run("digest override preserves digest and retains safe exec probe fallback", func(t *testing.T) {
		digestRepo := "myregistry.corp.internal/custom-sync@sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
		m := &mesheryv1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{Name: "meshery-meshsync", Namespace: "meshery"},
			Spec: mesheryv1alpha1.MeshSyncSpec{
				Repository: digestRepo,
			},
		}
		obj := GetServerObject(m, "")
		deploy := obj.(*appsv1.Deployment)
		c := deploy.Spec.Template.Spec.Containers[0]
		if c.Image != digestRepo {
			t.Errorf("image = %q, want %q", c.Image, digestRepo)
		}
		if c.ImagePullPolicy != corev1.PullIfNotPresent {
			t.Errorf("pullPolicy = %v, want IfNotPresent", c.ImagePullPolicy)
		}
		if c.ReadinessProbe != nil {
			t.Error("expected no readiness probe for digest without version")
		}
		if c.LivenessProbe == nil || c.LivenessProbe.Exec == nil {
			t.Error("expected safe exec liveness probe fallback for digest without version")
		}
	})
}

// TestDefaultMeshSyncVersionIsPinned is the regression gate for the class of
// broken install that motivated the pin: a moving channel tag as the default
// re-points under running clusters, so an install that worked yesterday pulls
// a different MeshSync tomorrow. It fails against the previous
// "stable-latest" default and against any future moving tag.
func TestDefaultMeshSyncVersionIsPinned(t *testing.T) {
	if utils.MovingTag(defaultMeshSyncVersion) {
		t.Fatalf("defaultMeshSyncVersion = %q is a moving channel tag; pin a released version", defaultMeshSyncVersion)
	}
	v, err := semver.NewVersion(defaultMeshSyncVersion)
	if err != nil {
		t.Fatalf("defaultMeshSyncVersion = %q is not a semantic version: %v", defaultMeshSyncVersion, err)
	}
	// Registry convention: meshery/meshsync publishes bare-semver image tags,
	// so a v-prefixed default would 404 at pull time.
	if strings.HasPrefix(defaultMeshSyncVersion, "v") {
		t.Errorf("defaultMeshSyncVersion = %q must be the bare form (%s); meshery/meshsync publishes no v-prefixed image tags", defaultMeshSyncVersion, v.String())
	}
	// The default must also be provably able to serve the health endpoints,
	// otherwise pinning it silently downgrades every default install to the
	// exec liveness fallback.
	if !servesHealthEndpoints(defaultMeshSyncVersion) {
		t.Errorf("defaultMeshSyncVersion = %q predates the /healthz and /readyz endpoints (%s)", defaultMeshSyncVersion, minHealthEndpointsVersion)
	}
}

// TestResolveVersion locks in the one place spec.version becomes a tag: unset
// falls back to the pinned default, an explicit pin is honoured, and a
// v-prefixed semver is normalised to the bare tag the registry publishes.
func TestResolveVersion(t *testing.T) {
	cases := map[string]string{
		"":              defaultMeshSyncVersion,
		tagPinned:       tagPinned,
		tagPinnedV:      tagPinned,
		tagEdgeLatest:   tagEdgeLatest,
		tagStableLatest: tagStableLatest,
		tagSha:          tagSha,
	}
	for in, want := range cases {
		if got := resolveVersion(in); got != want {
			t.Errorf("resolveVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestGetServerObjectImage proves the built Deployment - not just the helper -
// carries a pinned image and the matching pull policy by default, and honours
// spec.version when set.
func TestGetServerObjectImage(t *testing.T) {
	cases := []struct {
		version    string
		wantImage  string
		wantPolicy corev1.PullPolicy
	}{
		{"", meshsyncImageRepo + ":" + defaultMeshSyncVersion, corev1.PullIfNotPresent},
		{tagPinned, meshsyncImageRepo + ":1.0.2", corev1.PullIfNotPresent},
		{tagPinnedV, meshsyncImageRepo + ":1.0.2", corev1.PullIfNotPresent},
		{tagEdgeLatest, meshsyncImageRepo + ":edge-latest", corev1.PullAlways},
	}
	for _, tc := range cases {
		t.Run(versionLabel(tc.version), func(t *testing.T) {
			c := builtMeshSyncContainer(t, tc.version)
			if c.Image != tc.wantImage {
				t.Errorf("image = %q, want %q", c.Image, tc.wantImage)
			}
			if c.ImagePullPolicy != tc.wantPolicy {
				t.Errorf("imagePullPolicy = %q, want %q", c.ImagePullPolicy, tc.wantPolicy)
			}
		})
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
		tagPinnedV:      true,  // later patch
		"v1.1.0":        true,  // later minor
		"v2.0.0":        true,  // later major
		"v1.1.0-beta.1": true,  // pre-release that still postdates v1.0.1
		"v1.0.0":        false, // predates the endpoints
		"v0.8.26":       false, // predates the endpoints
		"v1.0.1-rc.1":   false, // pre-release of the boundary version, unprovable
		tagStableLatest: false, // moving channel tag, can't be proven
		tagEdgeLatest:   false, // moving channel tag, can't be proven
		"latest":        false, // moving tag
		"":              false, // empty -> template default image
		tagSha:          false, // commit-sha tag
	}
	for version, want := range cases {
		if got := servesHealthEndpoints(version); got != want {
			t.Errorf("servesHealthEndpoints(%q) = %v, want %v", version, got, want)
		}
	}
}

// TestGetServerObjectStrategy verifies the built Deployment specifies a
// RollingUpdate strategy with maxSurge: 0 and maxUnavailable: 1 so that
// upgrades terminate the existing pod before creating a replacement,
// preventing hostPort deadlock (#933).
func TestGetServerObjectStrategy(t *testing.T) {
	m := &mesheryv1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{Name: testObjName, Namespace: testObjName},
		Spec:       mesheryv1alpha1.MeshSyncSpec{Size: 1},
	}
	dep, ok := GetServerObject(m, "").(*appsv1.Deployment)
	if !ok {
		t.Fatal("GetServerObject did not return a *appsv1.Deployment")
	}

	if dep.Spec.Strategy.Type != appsv1.RollingUpdateDeploymentStrategyType {
		t.Fatalf("strategy type = %q, want %q", dep.Spec.Strategy.Type, appsv1.RollingUpdateDeploymentStrategyType)
	}

	ru := dep.Spec.Strategy.RollingUpdate
	if ru == nil {
		t.Fatal("expected rollingUpdate strategy configuration, got nil")
	}
	if ru.MaxSurge == nil {
		t.Fatal("expected maxSurge to be set, got nil")
	}
	if got := ru.MaxSurge.IntValue(); got != 0 || ru.MaxSurge.Type != intstr.Int {
		t.Errorf("maxSurge = %+v (int %d), want 0 (type int)", ru.MaxSurge, got)
	}
	if ru.MaxUnavailable == nil {
		t.Fatal("expected maxUnavailable to be set, got nil")
	}
	if got := ru.MaxUnavailable.IntValue(); got != 1 || ru.MaxUnavailable.Type != intstr.Int {
		t.Errorf("maxUnavailable = %+v (int %d), want 1 (type int)", ru.MaxUnavailable, got)
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

	// The default is a pinned release past the endpoint boundary, so an unset
	// spec.version now resolves to a provable version and gets the HTTP probes
	// too - it no longer takes the unprovable fallback path.
	t.Run("pinned default uses httpGet liveness and readiness", func(t *testing.T) {
		c := builtMeshSyncContainer(t, "")
		assertHTTPGetProbe(t, "liveness", c.LivenessProbe, healthzPath)
		assertHTTPGetProbe(t, "readiness", c.ReadinessProbe, readyzPath)
	})

	// Explicitly pinned moving tags and pre-endpoint versions must still fall
	// back to the exec liveness probe with no readiness probe.
	for _, version := range []string{tagStableLatest, "v1.0.0"} {
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

// Cross-repo contract names: the meshery-operator Helm chart in meshery/meshery
// creates these same objects and sets MESHSYNC_SERVICE_ACCOUNT to the same value.
const (
	contractMeshSyncServiceAccount = "meshery-meshsync"
	contractMeshSyncClusterRole    = "meshery-meshsync-role"
	contractMeshSyncBinding        = "meshery-meshsync-rolebinding"
	contractOperatorClusterRole    = "meshery-operator-role"
)

func TestResolveServiceAccountName(t *testing.T) {
	for value, want := range map[string]string{
		"":                             LegacyServiceAccountName,
		contractMeshSyncServiceAccount: contractMeshSyncServiceAccount,
		"custom.sa-1":                  "custom.sa-1",
	} {
		got, err := ResolveServiceAccountName(value)
		if err != nil || got != want {
			t.Errorf("ResolveServiceAccountName(%q) = %q, %v; want %q", value, got, err, want)
		}
	}
	for _, value := range []string{"Meshery-MeshSync", "meshery_meshsync", " meshery-meshsync", "-leading-dash", strings.Repeat("a", 254)} {
		got, err := ResolveServiceAccountName(value)
		if got != "" {
			t.Errorf("ResolveServiceAccountName(%q) = %q; an invalid name must not be used", value, got)
		}
		assertMeshkitError(t, err, ErrInvalidServiceAccountNameCode, "MESHSYNC_SERVICE_ACCOUNT=")
	}
}

func TestGetServerObjectServiceAccount(t *testing.T) {
	m := &mesheryv1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{Name: testObjName, Namespace: testObjName},
		Spec:       mesheryv1alpha1.MeshSyncSpec{Size: 1},
	}
	cases := []struct {
		name string
		want string
		opts []ServerOption
	}{
		{name: "no option keeps the legacy default", want: LegacyServiceAccountName},
		{name: "empty name keeps the legacy default", opts: []ServerOption{WithServiceAccountName("")}, want: LegacyServiceAccountName},
		{name: "configured name is applied", opts: []ServerOption{WithServiceAccountName(contractMeshSyncServiceAccount)}, want: contractMeshSyncServiceAccount},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dep, ok := GetServerObject(m, "", tc.opts...).(*appsv1.Deployment)
			if !ok {
				t.Fatal("GetServerObject did not return a *appsv1.Deployment")
			}
			if got := dep.Spec.Template.Spec.ServiceAccountName; got != tc.want {
				t.Errorf("serviceAccountName = %q, want %q", got, tc.want)
			}
		})
	}
	if got := Deployment.Spec.Template.Spec.ServiceAccountName; got != LegacyServiceAccountName {
		t.Errorf("options must not mutate the shared template, which now names %q", got)
	}
}

// TestCheckHealthReportsReplicaFailure verifies a Deployment that cannot create
// its pods (here: a missing ServiceAccount) surfaces that reason in the health
// error, and so in the MeshSync Ready condition.
func TestCheckHealthReportsReplicaFailure(t *testing.T) {
	m := &mesheryv1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{Name: testObjName, Namespace: testObjName},
		Spec:       mesheryv1alpha1.MeshSyncSpec{Size: 1},
	}
	const reason = `pods "test-abc" is forbidden: error looking up service account test/meshery-meshsync: serviceaccount "meshery-meshsync" not found`
	unready := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: testObjName, Namespace: testObjName},
		Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{
			{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Message: "unrelated"},
			{Type: appsv1.DeploymentReplicaFailure, Status: corev1.ConditionTrue, Reason: "FailedCreate", Message: reason},
		}},
	}
	c := fake.NewClientBuilder().WithScheme(k8sscheme.Scheme).WithObjects(unready).Build()
	err := CheckHealth(context.Background(), m, c)
	assertMeshkitError(t, err, ErrMeshsyncReplicasNotReadyCode, "0 of 1 replicas ready: "+reason)
	if strings.Contains(err.Error(), "unrelated") {
		t.Errorf("only ReplicaFailure conditions belong in the health error, got: %s", err)
	}

	// A ReplicaFailure condition with no message (reason only) must not leave
	// a trailing ": " on the health error.
	blank := unready.DeepCopy()
	blank.Status.Conditions[1].Message = ""
	c = fake.NewClientBuilder().WithScheme(k8sscheme.Scheme).WithObjects(blank).Build()
	err = CheckHealth(context.Background(), m, c)
	assertMeshkitError(t, err, ErrMeshsyncReplicasNotReadyCode, "0 of 1 replicas ready")
	if strings.HasSuffix(err.Error(), ": ") {
		t.Errorf("an empty ReplicaFailure message must not leave a trailing ': ', got: %s", err)
	}

	ready := unready.DeepCopy()
	ready.ResourceVersion = ""
	ready.Status = appsv1.DeploymentStatus{ReadyReplicas: 1}
	c = fake.NewClientBuilder().WithScheme(k8sscheme.Scheme).WithObjects(ready).Build()
	if err := CheckHealth(context.Background(), m, c); err != nil {
		t.Fatalf("a ready Deployment must be healthy, got %v", err)
	}
}

// TestRenderedInstallManifestWiresMeshSyncServiceAccount guards the cross-repo
// contract in the rendered install manifest (config/manifests/default.yaml,
// kept in sync with config/ by the manifests-drift CI gate): the operator
// Deployment names a ServiceAccount through MESHSYNC_SERVICE_ACCOUNT, that
// ServiceAccount exists in the operator's namespace, it is bound to a read-only
// ClusterRole, and the operator's own role stays wildcard-free. kustomize once
// rendered the binding subject unprefixed here (binding nothing), so the test
// reads the rendered output rather than the source files.
func TestRenderedInstallManifestWiresMeshSyncServiceAccount(t *testing.T) {
	objects := renderedInstallManifest(t)

	operator := &appsv1.Deployment{}
	decodeRendered(t, objects, "Deployment", "meshery-operator", operator)
	serviceAccountName := renderedServiceAccountEnv(t, operator)

	serviceAccount := &corev1.ServiceAccount{}
	decodeRendered(t, objects, "ServiceAccount", serviceAccountName, serviceAccount)
	if serviceAccount.Namespace != operator.Namespace {
		t.Errorf("ServiceAccount %s is in namespace %q, want the operator's %q", serviceAccountName, serviceAccount.Namespace, operator.Namespace)
	}

	binding := &rbacv1.ClusterRoleBinding{}
	decodeRendered(t, objects, "ClusterRoleBinding", contractMeshSyncBinding, binding)
	if binding.RoleRef.Kind != "ClusterRole" || binding.RoleRef.Name != contractMeshSyncClusterRole {
		t.Errorf("%s roleRef = %+v, want ClusterRole %s", contractMeshSyncBinding, binding.RoleRef, contractMeshSyncClusterRole)
	}
	wantSubject := rbacv1.Subject{Kind: "ServiceAccount", Name: serviceAccountName, Namespace: operator.Namespace}
	if len(binding.Subjects) != 1 || binding.Subjects[0] != wantSubject {
		t.Errorf("%s subjects = %+v, want exactly %+v", contractMeshSyncBinding, binding.Subjects, wantSubject)
	}

	role := &rbacv1.ClusterRole{}
	decodeRendered(t, objects, "ClusterRole", contractMeshSyncClusterRole, role)
	assertReadEverythingWriteNothing(t, role)

	operatorRole := &rbacv1.ClusterRole{}
	decodeRendered(t, objects, "ClusterRole", contractOperatorClusterRole, operatorRole)
	for _, rule := range operatorRole.Rules {
		if slices.Contains(rule.APIGroups, "*") || slices.Contains(rule.Resources, "*") || slices.Contains(rule.Verbs, "*") {
			t.Errorf("%s must stay least-privilege, found wildcard rule %+v", contractOperatorClusterRole, rule)
		}
	}
}

// renderedServiceAccountEnv returns the operator's MESHSYNC_SERVICE_ACCOUNT,
// failing unless it is the contract value and resolves cleanly.
func renderedServiceAccountEnv(t *testing.T, operator *appsv1.Deployment) string {
	t.Helper()
	value := ""
	for _, container := range operator.Spec.Template.Spec.Containers {
		for _, e := range container.Env {
			if e.Name == ServiceAccountEnv {
				value = e.Value
			}
		}
	}
	if value != contractMeshSyncServiceAccount {
		t.Fatalf("operator Deployment %s = %q, want %q", ServiceAccountEnv, value, contractMeshSyncServiceAccount)
	}
	if resolved, err := ResolveServiceAccountName(value); err != nil || resolved != value {
		t.Fatalf("ResolveServiceAccountName(%q) = %q, %v", value, resolved, err)
	}
	return value
}

// assertReadEverythingWriteNothing checks MeshSync's role: read on every group
// and resource (it watches CRDs discovered at runtime), and no other verb.
func assertReadEverythingWriteNothing(t *testing.T, role *rbacv1.ClusterRole) {
	t.Helper()
	readsEverything := false
	for _, rule := range role.Rules {
		for _, verb := range rule.Verbs {
			if !slices.Contains([]string{"get", "list", "watch"}, verb) {
				t.Errorf("%s grants %q; MeshSync's role must stay read-only", role.Name, verb)
			}
		}
		if len(rule.NonResourceURLs) > 0 {
			t.Errorf("%s grants nonResourceURLs %v; MeshSync needs none", role.Name, rule.NonResourceURLs)
		}
		if slices.Contains(rule.APIGroups, "*") && slices.Contains(rule.Resources, "*") {
			readsEverything = true
		}
	}
	if !readsEverything {
		t.Errorf("%s must read every group and resource: MeshSync watches arbitrary CRDs discovered at runtime", role.Name)
	}
}

// renderedInstallManifest decodes every document of config/manifests/default.yaml.
func renderedInstallManifest(t *testing.T) []*unstructured.Unstructured {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "config", "manifests", "default.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var objects []*unstructured.Unstructured
	decoder := yaml.NewYAMLOrJSONDecoder(f, 4096)
	for {
		object := &unstructured.Unstructured{}
		if err := decoder.Decode(&object.Object); err != nil {
			if errors.Is(err, io.EOF) {
				return objects
			}
			t.Fatal(err)
		}
		if len(object.Object) > 0 {
			objects = append(objects, object)
		}
	}
}

// decodeRendered converts the single rendered object of kind/name into out.
func decodeRendered(t *testing.T, objects []*unstructured.Unstructured, kind, name string, out any) {
	t.Helper()
	var found *unstructured.Unstructured
	for _, object := range objects {
		if object.GetKind() == kind && object.GetName() == name {
			if found != nil {
				t.Fatalf("rendered manifest has more than one %s %q", kind, name)
			}
			found = object
		}
	}
	if found == nil {
		t.Fatalf("rendered manifest has no %s %q", kind, name)
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(found.Object, out); err != nil {
		t.Fatal(err)
	}
}
