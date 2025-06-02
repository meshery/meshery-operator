package broker

import (
	"context"
	"fmt"
	"net"
	neturl "net/url"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	utils "github.com/meshery/meshery-operator/pkg/utils"
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

// GetEndpoint returns those endpoints in the given service which match the selector.
func GetEndpoint(ctx context.Context, m *mesheryv1alpha1.Broker, client *kubernetes.Clientset, url string) error {

	var serviceObj *corev1.Service
	var err error
	var newUrl *neturl.URL
	var host string

	serviceObj, err = client.CoreV1().Services(m.ObjectMeta.Namespace).Get(ctx, m.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		return ErrGettingResource(err)
	}

	var nodePort, clusterPort int32
	endpoint := utils.Endpoint{}

	for _, port := range serviceObj.Spec.Ports {
		nodePort = port.NodePort
		clusterPort = port.Port
		if port.Name == "client" {
			break
		}
	}
	// get clusterip endpoint
	endpoint.Internal = &utils.HostPort{
		Address: serviceObj.Spec.ClusterIP,
		Port:    clusterPort,
	}
	// Initialize nodePort type endpoint
	endpoint.External = &utils.HostPort{
		Address: "localhost",
		Port:    nodePort,
	}
	if serviceObj.Status.Size() > 0 && serviceObj.Status.LoadBalancer.Size() > 0 && len(serviceObj.Status.LoadBalancer.Ingress) > 0 && serviceObj.Status.LoadBalancer.Ingress[0].Size() > 0 {
		if serviceObj.Status.LoadBalancer.Ingress[0].IP == "" {
			endpoint.External.Address = serviceObj.Status.LoadBalancer.Ingress[0].Hostname
			endpoint.External.Port = clusterPort
		} else if serviceObj.Status.LoadBalancer.Ingress[0].IP == serviceObj.Spec.ClusterIP || serviceObj.Status.LoadBalancer.Ingress[0].IP == "<pending>" {
			if url != "" {
				newUrl, err = neturl.Parse(url)
				if err != nil {
					return err
				}
				host, _, err = net.SplitHostPort(newUrl.Host)
				if err != nil {
					return err
				}
				endpoint.External.Address = host
				endpoint.External.Port = nodePort
			} else {
				endpoint.External.Address = serviceObj.Spec.ClusterIP
				endpoint.External.Port = clusterPort
			}
		} else {
			endpoint.External.Address = serviceObj.Status.LoadBalancer.Ingress[0].IP
			endpoint.External.Port = clusterPort
		}
	}
	// Service Type ClusterIP
	if endpoint.External.Port == 0 {
		endpoint.Internal = &utils.HostPort{}
	}
	// If external endpoint not reachable
	if !utils.TcpCheck(endpoint.External, &utils.MockOptions{}) && endpoint.External.Address != "localhost" {
		newUrl, err = neturl.Parse(url)
		if err != nil {
			return nil
		}
		host, _, err = net.SplitHostPort(newUrl.Host)
		if err != nil {
			return nil
		}
		// Set to APIServer host (For minikube specific clusters)
		endpoint.External.Address = host
		// If still unable to reach, change to resolve to clusterPort
		if !utils.TcpCheck(endpoint.External, &utils.MockOptions{}) && endpoint.External.Address != "localhost" {
			endpoint.External.Port = nodePort
			if !utils.TcpCheck(endpoint.External, &utils.MockOptions{}) {
				return ErrGettingEndpoint(fmt.Errorf("unable to connect to endpoint at %v", endpoint.External))
			}
		}
	}

	m.Status.Endpoint.External = fmt.Sprintf("%s:%d", endpoint.External.Address, endpoint.External.Port)
	m.Status.Endpoint.Internal = fmt.Sprintf("%s:%d", endpoint.Internal.Address, endpoint.Internal.Port)
	return nil
}
