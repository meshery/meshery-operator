package meshsync

import (
	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	ServerObject = "server-object"
)

type Object interface {
	runtime.Object
	metav1.Object
}

func GetObjects(m *mesheryv1alpha1.MeshSync) map[string]Object {
	return map[string]Object{
		ServerObject: getServerObject(m.ObjectMeta.Namespace, m.ObjectMeta.Name, m.Spec.Size),
	}
}

func getServerObject(namespace string, name string, replicas int32) Object {
	obj := Deployment
	obj.ObjectMeta.Namespace = namespace
	obj.ObjectMeta.Name = name
	obj.Spec.Replicas = &replicas
	return obj
}
