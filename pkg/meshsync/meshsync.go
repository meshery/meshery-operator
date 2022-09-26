package meshsync

import (
	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
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
		ServerObject: getServerObject(m.ObjectMeta.Namespace, m.ObjectMeta.Name, m.Spec.Size, m.Status.PublishingTo),
	}
}

func getServerObject(namespace, name string, replicas int32, url string) Object {
	var obj = &v1.Deployment{}
	Deployment.DeepCopyInto(obj)
	obj.ObjectMeta.Namespace = namespace
	obj.ObjectMeta.Name = name
	obj.Spec.Replicas = &replicas
	obj.Spec.Template.Spec.Containers[0].Env[0].Value = url // Set broker endpoint
	return obj
}
