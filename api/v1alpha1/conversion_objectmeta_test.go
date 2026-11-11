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
	"testing"

	v1alpha2 "github.com/meshery/meshery-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestBrokerConversionObjectMetaIsolation ensures converted objects do not share
// label/annotation maps with their source, in both conversion directions and for
// both kinds.
func TestBrokerConversionObjectMetaIsolation(t *testing.T) {
	t.Run("Broker ConvertTo", func(t *testing.T) {
		orig := &Broker{ObjectMeta: metav1.ObjectMeta{Name: testBrokerName, Labels: map[string]string{"env": "production"}}}
		hub := &v1alpha2.Broker{}
		if err := orig.ConvertTo(hub); err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}
		hub.Labels["env"] = "staging"
		if orig.Labels["env"] != "production" {
			t.Errorf("Leaked metadata mutation: orig.Labels['env'] = %q, want 'production'", orig.Labels["env"])
		}
	})

	t.Run("Broker ConvertFrom", func(t *testing.T) {
		hub := &v1alpha2.Broker{ObjectMeta: metav1.ObjectMeta{Name: testBrokerName, Labels: map[string]string{"env": "production"}}}
		spoke := &Broker{}
		if err := spoke.ConvertFrom(hub); err != nil {
			t.Fatalf("ConvertFrom failed: %v", err)
		}
		spoke.Labels["env"] = "staging"
		if hub.Labels["env"] != "production" {
			t.Errorf("Leaked metadata mutation: hub.Labels['env'] = %q, want 'production'", hub.Labels["env"])
		}
	})

	t.Run("MeshSync ConvertTo", func(t *testing.T) {
		orig := &MeshSync{ObjectMeta: metav1.ObjectMeta{Name: testBrokerName, Labels: map[string]string{"env": "production"}}}
		hub := &v1alpha2.MeshSync{}
		if err := orig.ConvertTo(hub); err != nil {
			t.Fatalf("ConvertTo failed: %v", err)
		}
		hub.Labels["env"] = "staging"
		if orig.Labels["env"] != "production" {
			t.Errorf("Leaked metadata mutation: orig.Labels['env'] = %q, want 'production'", orig.Labels["env"])
		}
	})

	t.Run("MeshSync ConvertFrom", func(t *testing.T) {
		hub := &v1alpha2.MeshSync{ObjectMeta: metav1.ObjectMeta{Name: testBrokerName, Labels: map[string]string{"env": "production"}}}
		spoke := &MeshSync{}
		if err := spoke.ConvertFrom(hub); err != nil {
			t.Fatalf("ConvertFrom failed: %v", err)
		}
		spoke.Labels["env"] = "staging"
		if hub.Labels["env"] != "production" {
			t.Errorf("Leaked metadata mutation: hub.Labels['env'] = %q, want 'production'", hub.Labels["env"])
		}
	})
}

// TestConversionNilAndWrongTypeReturnError ensures nil, typed-nil, and wrong-type
// arguments produce errors instead of panicking.
func TestConversionNilAndWrongTypeReturnError(t *testing.T) {
	b := &Broker{}
	m := &MeshSync{}

	cases := map[string]func() error{
		"Broker ConvertTo nil":           func() error { return b.ConvertTo(nil) },
		"Broker ConvertTo typed nil":     func() error { return b.ConvertTo((*v1alpha2.Broker)(nil)) },
		"Broker ConvertTo wrong type":    func() error { return b.ConvertTo(&v1alpha2.MeshSync{}) },
		"Broker ConvertFrom nil":         func() error { return b.ConvertFrom(nil) },
		"Broker ConvertFrom typed nil":   func() error { return b.ConvertFrom((*v1alpha2.Broker)(nil)) },
		"Broker ConvertFrom wrong type":  func() error { return b.ConvertFrom(&v1alpha2.MeshSync{}) },
		"MeshSync ConvertTo nil":         func() error { return m.ConvertTo(nil) },
		"MeshSync ConvertTo typed nil":   func() error { return m.ConvertTo((*v1alpha2.MeshSync)(nil)) },
		"MeshSync ConvertFrom nil":       func() error { return m.ConvertFrom(nil) },
		"MeshSync ConvertFrom typed nil": func() error { return m.ConvertFrom((*v1alpha2.MeshSync)(nil)) },
	}

	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked: %v", r)
				}
			}()
			if err := call(); err == nil {
				t.Errorf("expected an error, got nil")
			}
		})
	}
}
