package broker

import (
	"context"
	"fmt"

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

func getServerObject(namespace, name string, replicas int32) Object {
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

func GetEndpoint(ctx context.Context, m *mesheryv1alpha1.Broker, client *kubernetes.Clientset) error {
	obj, err := client.CoreV1().Services(m.ObjectMeta.Namespace).Get(ctx, m.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		return ErrGettingResource(err)
	}

	// // To be upgraded for client-go 0.20+
	// if obj.Status.Conditions[0].Status == corev1.ConditionFalse || obj.Status.Conditions[0].Status == corev1.ConditionUnknown {
	// 	return ErrConditionFalse(obj.Status.Conditions[0].Reason)
	// }

	// m.Status.Endpoint = fmt.Sprintf("http://%s:%d", obj.Status.LoadBalancer.Ingress[0].IP, obj.Status.LoadBalancer.Ingress[0].Ports[0].Port)

	m.Status.Endpoint = fmt.Sprintf("http://%s:4222", obj.Status.LoadBalancer.Ingress[0].IP)
	return nil
}
