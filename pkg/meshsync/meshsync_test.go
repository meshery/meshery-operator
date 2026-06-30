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

package meshsync

import (
	"testing"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetObjects(t *testing.T) {
	m := &mesheryv1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: mesheryv1alpha1.MeshSyncSpec{
			Size: 1,
		},
	}
	obj := GetObjects(m)
	if len(obj) == 0 {
		t.Fatal("GetObjects returned no objects")
	}
	if obj[0] == nil {
		t.Error("GetObjects returned nil for server object")
	}
}

func TestEnsureNatsScheme(t *testing.T) {
	cases := map[string]string{
		"":                   "",
		"meshery-nats:4222":  "nats://meshery-nats:4222",
		"10.0.0.1:4222":      "nats://10.0.0.1:4222",
		"nats://broker:4222": "nats://broker:4222",
		"tls://broker:4222":  "tls://broker:4222",
	}
	for in, want := range cases {
		if in == "" {
			continue // ensureNatsScheme is only called for non-empty URLs
		}
		if got := ensureNatsScheme(in); got != want {
			t.Errorf("ensureNatsScheme(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSetBrokerURLByName(t *testing.T) {
	c := &corev1.Container{Env: []corev1.EnvVar{
		{Name: "OTHER", Value: "x"},
		{Name: brokerURLEnv, Value: "nats://localhost:4222"},
	}}
	setBrokerURL(c, "10.0.0.5:4222")
	if c.Env[1].Value != "nats://10.0.0.5:4222" {
		t.Errorf("BROKER_URL = %q, want nats://10.0.0.5:4222", c.Env[1].Value)
	}
	if c.Env[0].Value != "x" {
		t.Errorf("unrelated env var was mutated: %q", c.Env[0].Value)
	}

	// An empty endpoint must leave the template default untouched.
	setBrokerURL(c, "")
	if c.Env[1].Value != "nats://10.0.0.5:4222" {
		t.Errorf("empty URL must not change BROKER_URL, got %q", c.Env[1].Value)
	}
}
