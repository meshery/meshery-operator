package broker

import (
	"context"
	"fmt"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	mesherykube "github.com/layer5io/meshkit/utils/kubernetes"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
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

func CheckHealth(ctx context.Context, m *mesheryv1alpha1.Broker, client *kubernetes.Clientset) error {
	obj, err := client.AppsV1().StatefulSets(m.ObjectMeta.Namespace).Get(ctx, m.ObjectMeta.Name, metav1.GetOptions{})
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

func GetEndpoint(ctx context.Context, m *mesheryv1alpha1.Broker, client *kubernetes.Clientset, url string) error {
	endpoint, err := mesherykube.GetServiceEndpoint(context.TODO(), client, &mesherykube.ServiceOptions{
		Name:         m.ObjectMeta.Name,
		Namespace:    m.ObjectMeta.Namespace,
		PortSelector: "client",
		APIServerURL: url,
	})
	if err != nil {
		return ErrGettingEndpoint(err)
	}

	m.Status.Endpoint.External = fmt.Sprintf("%s:%d", endpoint.External.Address, endpoint.External.Port)
	m.Status.Endpoint.Internal = fmt.Sprintf("%s:%d", endpoint.Internal.Address, endpoint.Internal.Port)
	return nil
}
