/*
Copyright 2023 Layer5, Inc.

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
	mesheryv1beta1 "github.com/layer5io/meshery-operator/api/v1beta1"
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

func GetObjects(m *mesheryv1beta1.MeshSync) map[string]Object {
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
