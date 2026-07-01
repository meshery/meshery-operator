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

package v1alpha1

import (
	"reflect"
	"testing"

	v1alpha2 "github.com/meshery/meshery-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBrokerConversionRoundTrip(t *testing.T) {
	lbClass := "internal"
	orig := &Broker{
		ObjectMeta: metav1.ObjectMeta{Name: "meshery-broker", Namespace: "meshery", Labels: map[string]string{"app": "meshery"}},
		Spec: BrokerSpec{
			Version: "2.14.2-alpine",
			Size:    3,
			Service: BrokerServiceSpec{
				Type:                     corev1.ServiceTypeLoadBalancer,
				Annotations:              map[string]string{"a": "b"},
				LoadBalancerClass:        &lbClass,
				LoadBalancerSourceRanges: []string{"10.0.0.0/8"},
				ExternalEndpointOverride: "ext.example.com:4222",
			},
		},
		Status: BrokerStatus{
			Endpoint:   Endpoint{Internal: "10.0.0.1:4222", External: "1.2.3.4:4222"},
			Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, Reason: "ok"}},
		},
	}

	hub := &v1alpha2.Broker{}
	if err := orig.ConvertTo(hub); err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}
	if hub.Spec.Size != 3 || hub.Spec.Service.Type != corev1.ServiceTypeLoadBalancer || hub.Status.Endpoint.Internal != "10.0.0.1:4222" {
		t.Fatalf("hub not populated correctly: %+v", hub.Spec)
	}

	back := &Broker{}
	if err := back.ConvertFrom(hub); err != nil {
		t.Fatalf("ConvertFrom: %v", err)
	}
	if !reflect.DeepEqual(orig, back) {
		t.Errorf("round-trip mismatch:\n orig=%+v\n back=%+v", orig, back)
	}
}

func TestMeshSyncConversionRoundTrip(t *testing.T) {
	orig := &MeshSync{
		ObjectMeta: metav1.ObjectMeta{Name: "meshery-meshsync", Namespace: "meshery"},
		Spec: MeshSyncSpec{
			Size:    2,
			Version: "stable-latest",
			Broker:  MeshsyncBroker{Native: NativeMeshsyncBroker{Name: "meshery-broker", Namespace: "meshery"}},
			WatchList: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "meshery-meshsync-watch"},
				Data:       map[string]string{"whitelist": "[]"},
			},
		},
		Status: MeshSyncStatus{
			PublishingTo: "nats://10.0.0.1:4222",
			Conditions:   []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, Reason: "ok"}},
		},
	}

	hub := &v1alpha2.MeshSync{}
	if err := orig.ConvertTo(hub); err != nil {
		t.Fatalf("ConvertTo: %v", err)
	}
	if hub.Spec.Broker.Native.Name != "meshery-broker" || hub.Spec.WatchList.Data["whitelist"] != "[]" {
		t.Fatalf("hub not populated correctly: %+v", hub.Spec)
	}

	back := &MeshSync{}
	if err := back.ConvertFrom(hub); err != nil {
		t.Fatalf("ConvertFrom: %v", err)
	}
	if !reflect.DeepEqual(orig, back) {
		t.Errorf("round-trip mismatch:\n orig=%+v\n back=%+v", orig, back)
	}
}
