package broker

import (
	"context"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	meshkitkube "github.com/meshery/meshkit/utils/kubernetes"
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

func GetObjects(m *mesheryv1alpha1.Broker) map[string]Object {
	return map[string]Object{
		ServerConfig:  getServerConfig(),
		AccountConfig: getAccountConfig(),
		ServerObject:  getServerObject(m.ObjectMeta.Namespace, m.ObjectMeta.Name, m.Spec.Size),
		ServiceObject: getServiceObject(m.ObjectMeta.Namespace, m.ObjectMeta.Name),
	}
}

func getServerObject(namespace, name string, replicas int32) Object {
	var obj = &v1.StatefulSet{}
	StatefulSet.DeepCopyInto(obj)
	obj.ObjectMeta.Namespace = namespace
	obj.ObjectMeta.Name = name
	obj.Spec.Replicas = &replicas
	return obj
}

func getServiceObject(namespace, name string) Object {
	var obj = &corev1.Service{}
	Service.DeepCopyInto(obj)
	obj.ObjectMeta.Name = name
	obj.ObjectMeta.Namespace = namespace
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

func CheckHealth(ctx context.Context, m *mesheryv1alpha1.Broker, client client.Client) error {
	obj := &v1.StatefulSet{}
	err := client.Get(ctx, types.NamespacedName{Name: m.ObjectMeta.Name, Namespace: m.ObjectMeta.Namespace}, obj)
	if err != nil {
		return ErrGettingResource(err)
	}

	if obj.Status.Replicas != obj.Status.ReadyReplicas {
		if len(obj.Status.Conditions) > 0 {
			return ErrReplicasNotReady(obj.Status.Conditions[0].Reason)
		}
		return ErrReplicasNotReady("Condition Unknown")
	}

	if len(obj.Status.Conditions) > 0 && (obj.Status.Conditions[0].Status == corev1.ConditionFalse || obj.Status.Conditions[0].Status == corev1.ConditionUnknown) {
		return ErrConditionFalse(obj.Status.Conditions[0].Reason)
	}

	return nil
}

// GetEndpoint returns those endpoints in the given service which match the selector.
func GetEndpoint(ctx context.Context, m *mesheryv1alpha1.Broker, client client.Client, url string) error {
	serviceObj := &corev1.Service{}
	err := client.Get(ctx, types.NamespacedName{Name: m.ObjectMeta.Name, Namespace: m.ObjectMeta.Namespace}, serviceObj)
	if err != nil {
		return ErrGettingResource(err)
	}

	opts := &meshkitkube.ServiceOptions{
		Name:         m.ObjectMeta.Name,
		Namespace:    m.ObjectMeta.Namespace,
		PortSelector: "client",
		APIServerURL: url,
		WorkerNodeIP: "localhost",
	}

	endpoint, err := meshkitkube.GetEndpoint(ctx, opts, serviceObj)
	if err != nil {
		return ErrGettingEndpoint(err)
	}

	// Set the broker status endpoints
	if endpoint.External != nil {
		m.Status.Endpoint.External = endpoint.External.String()
	}
	if endpoint.Internal != nil {
		m.Status.Endpoint.Internal = endpoint.Internal.String()
	}

	return nil
}
