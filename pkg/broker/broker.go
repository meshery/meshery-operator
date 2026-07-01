package broker

import (
	"context"
	"fmt"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServerConfig  = "server-config"
	AccountConfig = "account-config"
	ServerObject  = "server-object"
	ServiceObject = "service-object"
)

type Object interface {
	runtime.Object
	metav1.Object
}

// GetObjects returns the broker-owned objects in a deterministic order:
// ConfigMaps and Service before the StatefulSet, so the workload's config and
// (clusterIP-bearing) Service exist before endpoint derivation runs. A slice —
// not a map — guarantees that order on every reconcile (WS-3 §4.3 #16).
func GetObjects(m *mesheryv1alpha1.Broker) []Object {
	return []Object{
		getServerConfig(),
		getAccountConfig(),
		getServiceObject(m.Namespace, m.Name),
		getServerObject(m.Namespace, m.Name, desiredReplicas(m)),
	}
}

// desiredReplicas defaults an unset spec.size to one replica so an omitted
// size never applies a zero-replica StatefulSet that CheckHealth (which
// expects one ready replica) would report unhealthy forever.
func desiredReplicas(m *mesheryv1alpha1.Broker) int32 {
	if m.Spec.Size > 0 {
		return m.Spec.Size
	}
	return 1
}

func getServerObject(namespace, name string, replicas int32) Object {
	var obj = &v1.StatefulSet{}
	StatefulSet.DeepCopyInto(obj)
	obj.Namespace = namespace
	obj.Name = name
	obj.Spec.Replicas = &replicas
	return obj
}

func getServiceObject(namespace, name string) Object {
	var obj = &corev1.Service{}
	Service.DeepCopyInto(obj)
	obj.Name = name
	obj.Namespace = namespace
	return obj
}

func getServerConfig() Object {
	var obj = &corev1.ConfigMap{}
	NatsConfigMap.DeepCopyInto(obj)
	return obj
}

func getAccountConfig() Object {
	var obj = &corev1.ConfigMap{}
	AccountsConfigMap.DeepCopyInto(obj)
	return obj
}

// CheckHealth reports whether the broker StatefulSet has reached its desired
// ready replica count. StatefulSets do not populate status.conditions reliably,
// so ReadyReplicas is the authoritative signal (WS-3 §4.3 #17).
func CheckHealth(ctx context.Context, m *mesheryv1alpha1.Broker, c client.Client) error {
	obj := &v1.StatefulSet{}
	if err := c.Get(ctx, types.NamespacedName{Name: m.Name, Namespace: m.Namespace}, obj); err != nil {
		return ErrGettingBrokerResource(err)
	}

	desired := desiredReplicas(m)
	if obj.Status.ReadyReplicas != desired {
		return ErrBrokerReplicasNotReady(fmt.Sprintf("%d of %d replicas ready", obj.Status.ReadyReplicas, desired))
	}
	return nil
}

// GetEndpoint derives the broker endpoints from the Service and writes them to
// the Broker status. Derivation is pure and non-blocking: it reads only the
// Service spec/status (no TCP dials), so it never stalls the reconcile queue
// (WS-3 §4.3 #14, WS-6). Addresses are stored as host:port; the nats:// scheme
// is applied where the endpoint is consumed (MeshSync BROKER_URL injection).
func GetEndpoint(ctx context.Context, m *mesheryv1alpha1.Broker, c client.Client, apiServerURL string) error {
	serviceObj := &corev1.Service{}
	if err := c.Get(ctx, types.NamespacedName{Name: m.Name, Namespace: m.Namespace}, serviceObj); err != nil {
		return ErrGettingBrokerResource(err)
	}

	internal, external, _ := DeriveEndpoint(serviceObj, apiServerURL)
	m.Status.Endpoint.Internal = internal
	m.Status.Endpoint.External = external
	return nil
}
