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

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
)

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
		setBrokerURL(&obj.Spec.Template.Spec.Containers[0], m.Status.PublishingTo, tokenSecret)
	}
	return obj
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
