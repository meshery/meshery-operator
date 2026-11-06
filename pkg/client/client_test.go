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

package client

import (
	"testing"

	"k8s.io/client-go/rest"
)

// New must not modify the caller's rest.Config: callers pass a config shared
// with their other Kubernetes clients, and often call New repeatedly from
// concurrent status checks.
func TestNewDoesNotMutateConfig(t *testing.T) {
	config := &rest.Config{Host: "https://127.0.0.1:6443"}

	cs, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if cs == nil || cs.CoreV1Alpha1() == nil {
		t.Fatal("New() returned a nil client")
	}

	if config.GroupVersion != nil {
		t.Errorf("config.GroupVersion = %v, want nil", config.GroupVersion)
	}
	if config.APIPath != "" {
		t.Errorf("config.APIPath = %q, want empty", config.APIPath)
	}
	if config.ContentType != "" {
		t.Errorf("config.ContentType = %q, want empty", config.ContentType)
	}
	if config.NegotiatedSerializer != nil {
		t.Errorf("config.NegotiatedSerializer = %v, want nil", config.NegotiatedSerializer)
	}
}
