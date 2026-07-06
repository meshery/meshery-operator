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
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServerObject = "server-object"
	// brokerURLEnv is the env var the MeshSync container reads its broker
	// connection string from.
	brokerURLEnv = "BROKER_URL"
	// natsTokenEnv carries the NATS auth token, sourced from the broker's auth
	// Secret via secretKeyRef; BROKER_URL splices it in through Kubernetes
	// $(VAR) expansion so the credential never appears verbatim in the
	// Deployment spec or the CR status.
	natsTokenEnv = "NATS_TOKEN"
	// natsTokenKey is the key inside the auth Secret (matches the Secret the
	// broker controller generates, pkg/broker BuildAuthSecret).
	natsTokenKey = "token"
	// defaultBrokerURL is the template placeholder BROKER_URL, left in place
	// until the operator derives the real broker endpoint.
	defaultBrokerURL = "nats://localhost:4222"

	// minHealthEndpointsVersion is the first released MeshSync version that
	// serves the /healthz and /readyz HTTP endpoints on the client port
	// (meshsync v1.0.1). Images at or above it get httpGet probes; older or
	// unprovable versions keep the exec liveness baked into the template.
	minHealthEndpointsVersion = "v1.0.1"
	// healthzPath is MeshSync's liveness endpoint: always 200 while the process
	// is alive. readyzPath is its readiness endpoint: 503 until MeshSync has
	// connected to the broker at least once, then a permanent 200.
	healthzPath = "/healthz"
	readyzPath  = "/readyz"
)

// minHealthEndpoints is the parsed form of minHealthEndpointsVersion, compared
// against spec.version to decide the probe strategy.
var minHealthEndpoints = semver.MustParse(minHealthEndpointsVersion)

type Object interface {
	runtime.Object
	metav1.Object
}

// GetObjects returns the MeshSync-owned objects as a deterministic slice.
// tokenSecret names the NATS auth Secret in the MeshSync's namespace, or "" when
// the broker runs without token auth.
func GetObjects(m *mesheryv1alpha1.MeshSync, tokenSecret string) []Object {
	return []Object{GetServerObject(m, tokenSecret)}
}

// GetServerObject builds the MeshSync Deployment for the given CR, injecting the
// broker URL by env name (not position) and normalising it to a nats:// scheme
// (WS-3 §4.3 #18). tokenSecret names the NATS auth Secret in the MeshSync's
// namespace ("" when the broker runs without token auth).
func GetServerObject(m *mesheryv1alpha1.MeshSync, tokenSecret string) Object {
	obj := &v1.Deployment{}
	Deployment.DeepCopyInto(obj)
	obj.Namespace = m.Namespace
	obj.Name = m.Name
	size := desiredReplicas(m)
	obj.Spec.Replicas = &size
	if len(obj.Spec.Template.Spec.Containers) > 0 {
		applyVersion(&obj.Spec.Template.Spec.Containers[0], m.Spec.Version)
		applyProbes(&obj.Spec.Template.Spec.Containers[0], m.Spec.Version)
		setBrokerURL(&obj.Spec.Template.Spec.Containers[0], m.Status.PublishingTo, tokenSecret)
	}
	return obj
}

// applyVersion maps spec.version onto the MeshSync image tag. Moving tags
// (…-latest) keep PullAlways so clusters track the channel; pinned tags switch
// to IfNotPresent so side-loaded images (kind) and air-gapped clusters work.
func applyVersion(c *corev1.Container, version string) {
	if version == "" {
		return
	}
	c.Image = meshsyncImageRepo + ":" + version
	if strings.HasSuffix(version, "-latest") {
		c.ImagePullPolicy = corev1.PullAlways
	} else {
		c.ImagePullPolicy = corev1.PullIfNotPresent
	}
}

// applyProbes selects the container's health probes from the deployed MeshSync
// version. Versions that serve the HTTP health endpoints (>= v1.0.1) get a
// cheap httpGet /healthz liveness probe (no binary fork, so no false restarts
// under the CPU pressure of the initial full-cluster sync) plus an httpGet
// /readyz readiness probe that holds the pod NotReady until MeshSync has
// connected to the broker once. Every other version - older images, moving
// channel tags such as the default stable-latest, and unparseable tags - keeps
// the exec liveness baked into the template and gets no readiness probe:
// probing an image that serves nothing on the client port over HTTP would
// connection-refuse and crashloop an otherwise-healthy pod (version skew,
// since spec.version is user-settable).
//
// /readyz is a one-shot latch (it never flips back to 503 once connected), so
// the readiness probe cannot wedge a running pod on a transient broker blip -
// which is why it is safe here even though an earlier exec-based readiness
// probe was removed for stalling rollout on CPU-starved nodes.
func applyProbes(c *corev1.Container, version string) {
	if !servesHealthEndpoints(version) {
		// Keep the template's exec liveness; attach no readiness probe.
		return
	}
	port := intstr.FromString(clientPortName)
	c.LivenessProbe = &corev1.Probe{
		InitialDelaySeconds: 10,
		PeriodSeconds:       30,
		TimeoutSeconds:      5,
		FailureThreshold:    5,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{Path: healthzPath, Port: port},
		},
	}
	c.ReadinessProbe = &corev1.Probe{
		InitialDelaySeconds: 5,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    3,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{Path: readyzPath, Port: port},
		},
	}
}

// servesHealthEndpoints reports whether spec.version is a pinned semantic
// version at or above minHealthEndpointsVersion, i.e. an image known to serve
// /healthz and /readyz. Moving channel tags (stable-latest, edge-latest, ""),
// commit-sha tags, and versions predating the endpoints are unparseable or
// lower and return false, so their pods stay on the exec liveness probe. The
// gate is deliberately conservative: an unprovable version - including a
// pre-release of the boundary version such as v1.0.1-rc.1, which sorts below
// v1.0.1 and may predate the endpoint commit - is treated as lacking the
// endpoints rather than risking an httpGet crashloop.
func servesHealthEndpoints(version string) bool {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}
	return !v.LessThan(minHealthEndpoints)
}

// setBrokerURL sets the BROKER_URL env var by name, leaving the template default
// in place when no endpoint has been derived yet. With a tokenSecret, the token
// is referenced as $(NATS_TOKEN) userinfo and NATS_TOKEN is sourced from the
// Secret; NATS_TOKEN is placed before BROKER_URL because $(VAR) references only
// expand when VAR is defined earlier in the env list.
func setBrokerURL(container *corev1.Container, rawURL, tokenSecret string) {
	if rawURL == "" {
		return
	}
	value := ensureNatsScheme(rawURL)
	if tokenSecret == "" {
		setEnvValue(container, brokerURLEnv, value)
		return
	}

	container.Env = removeEnv(container.Env, natsTokenEnv, brokerURLEnv)
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name: natsTokenEnv,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: tokenSecret},
					Key:                  natsTokenKey,
				},
			},
		},
		corev1.EnvVar{Name: brokerURLEnv, Value: withTokenUserinfo(value)},
	)
}

// setEnvValue sets a literal-valued env var by name, appending it when absent.
func setEnvValue(container *corev1.Container, name, value string) {
	for i := range container.Env {
		if container.Env[i].Name == name {
			container.Env[i].Value = value
			container.Env[i].ValueFrom = nil
			return
		}
	}
	container.Env = append(container.Env, corev1.EnvVar{Name: name, Value: value})
}

// removeEnv returns env without the named entries.
func removeEnv(env []corev1.EnvVar, names ...string) []corev1.EnvVar {
	out := make([]corev1.EnvVar, 0, len(env))
	for _, e := range env {
		if !slices.Contains(names, e.Name) {
			out = append(out, e)
		}
	}
	return out
}

// withTokenUserinfo splices the $(NATS_TOKEN) reference into the URL as
// userinfo: nats://host:port -> nats://$(NATS_TOKEN)@host:port. URLs already
// carrying userinfo are left untouched.
func withTokenUserinfo(u string) string {
	i := strings.Index(u, "://")
	if i < 0 {
		return u
	}
	rest := u[i+3:]
	if strings.Contains(rest, "@") {
		return u
	}
	return u[:i+3] + "$(" + natsTokenEnv + ")@" + rest
}

// desiredReplicas defaults an unset spec.size to one replica so an omitted
// size never applies a zero-replica Deployment that CheckHealth (which
// expects one ready replica) would report unhealthy forever.
func desiredReplicas(m *mesheryv1alpha1.MeshSync) int32 {
	if m.Spec.Size > 0 {
		return m.Spec.Size
	}
	return 1
}

// ensureNatsScheme prefixes a bare host:port with nats://, while leaving an
// already-schemed URL (nats://, tls://, …) untouched.
func ensureNatsScheme(raw string) string {
	if strings.Contains(raw, "://") {
		return raw
	}
	return "nats://" + raw
}

// CheckHealth reports whether the MeshSync Deployment has reached its desired
// ready replica count (ReadyReplicas is the authoritative signal — WS-3 §4.3 #17).
func CheckHealth(ctx context.Context, m *mesheryv1alpha1.MeshSync, c client.Client) error {
	obj := &v1.Deployment{}
	if err := c.Get(ctx, types.NamespacedName{Name: m.Name, Namespace: m.Namespace}, obj); err != nil {
		return ErrGettingMeshsyncResource(err)
	}

	desired := desiredReplicas(m)
	if obj.Status.ReadyReplicas != desired {
		return ErrMeshsyncReplicasNotReady(fmt.Sprintf("%d of %d replicas ready", obj.Status.ReadyReplicas, desired))
	}
	return nil
}
