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
)

type Object interface {
	runtime.Object
	metav1.Object
}

// GetObjects returns the MeshSync-owned objects as a deterministic slice.
func GetObjects(m *mesheryv1alpha1.MeshSync) []Object {
	return []Object{GetServerObject(m)}
}

// GetServerObject builds the MeshSync Deployment for the given CR, injecting the
// broker URL by env name (not position) and normalising it to a nats:// scheme
// (WS-3 §4.3 #18).
func GetServerObject(m *mesheryv1alpha1.MeshSync) Object {
	obj := &v1.Deployment{}
	Deployment.DeepCopyInto(obj)
	obj.Namespace = m.Namespace
	obj.Name = m.Name
	size := desiredReplicas(m)
	obj.Spec.Replicas = &size
	if len(obj.Spec.Template.Spec.Containers) > 0 {
		setBrokerURL(&obj.Spec.Template.Spec.Containers[0], m.Status.PublishingTo)
	}
	return obj
}

// setBrokerURL sets the BROKER_URL env var by name, leaving the template default
// in place when no endpoint has been derived yet.
func setBrokerURL(container *corev1.Container, rawURL string) {
	if rawURL == "" {
		return
	}
	value := ensureNatsScheme(rawURL)
	for i := range container.Env {
		if container.Env[i].Name == brokerURLEnv {
			container.Env[i].Value = value
			return
		}
	}
	container.Env = append(container.Env, corev1.EnvVar{Name: brokerURLEnv, Value: value})
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
