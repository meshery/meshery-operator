package broker

import (
	"context"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
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
		ServiceObject: getServiceObject(),
	}
}

func getServerObject(namespace string, name string, replicas int32) Object {
	obj := StatefulSet
	obj.ObjectMeta.Namespace = namespace
	obj.ObjectMeta.Name = name
	obj.Spec.Replicas = &replicas
	return obj
}

func getServiceObject() Object {
	obj := Service
	return obj
}

func getServerConfig() Object {
	obj := NatsConfigMap
	return obj
}

func getAccountConfig() Object {
	obj := AccountsConfigMap
	return obj
}

func CheckHealth(ctx context.Context, m *mesheryv1alpha1.Broker, client *kubernetes.Clientset) error {
	obj, err := client.AppsV1().StatefulSets(m.ObjectMeta.Namespace).Get(ctx, m.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		return ErrGettingResource(err)
	}

	if obj.Status.Replicas != obj.Status.ReadyReplicas {
		return ErrReplicasNotReady(obj.Status.Conditions[0].Reason)
	}

	if obj.Status.Conditions[0].Status == corev1.ConditionFalse || obj.Status.Conditions[0].Status == corev1.ConditionUnknown {
		return ErrConditionFalse(obj.Status.Conditions[0].Reason)
	}

	return nil
}

func GetEndpoint(ctx context.Context, m *mesheryv1alpha1.Broker, client *kubernetes.Clientset) error {
	obj, err := client.CoreV1().Services(m.ObjectMeta.Namespace).Get(ctx, m.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		return ErrGettingResource(err)
	}

	// if obj.Status.Conditions[0].Status == corev1.ConditionFalse || obj.Status.Conditions[0].Status == corev1.ConditionUnknown {
	// 	return ErrConditionFalse(obj.Status.Conditions[0].Reason)
	// }

	m.Status.Endpoint = obj.Status.LoadBalancer.Ingress[0].IP
	return nil
}
