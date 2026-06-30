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
package controllers

import (
	"errors"
	"strings"
	"testing"

	meshkiterrors "github.com/meshery/meshkit/errors"
)

// assertMeshkitError verifies a constructor returns a MeshKit structured error
// carrying the expected code, Alert severity, and the underlying cause.
func assertMeshkitError(t *testing.T, err error, wantCode, wantCause string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := meshkiterrors.GetCode(err); got != wantCode {
		t.Errorf("expected code %q, got %q", wantCode, got)
	}
	if got := meshkiterrors.GetSeverity(err); got != meshkiterrors.Alert {
		t.Errorf("expected severity Alert (%d), got %d", meshkiterrors.Severity(meshkiterrors.Alert), got)
	}
	if wantCause != "" && !strings.Contains(err.Error(), wantCause) {
		t.Errorf("expected error to carry cause %q, got: %s", wantCause, err.Error())
	}
}

func TestErrGetMeshsync(t *testing.T) {
	assertMeshkitError(t, ErrGetMeshsync(errors.New("boom")), ErrGetMeshsyncCode, "boom")
}

func TestErrCreateMeshsync(t *testing.T) {
	assertMeshkitError(t, ErrCreateMeshsync(errors.New("boom")), ErrCreateMeshsyncCode, "boom")
}

func TestErrDeleteMeshsync(t *testing.T) {
	assertMeshkitError(t, ErrDeleteMeshsync(errors.New("boom")), ErrDeleteMeshsyncCode, "boom")
}

func TestErrReconcileMeshsync(t *testing.T) {
	assertMeshkitError(t, ErrReconcileMeshsync(errors.New("boom")), ErrReconcileMeshsyncCode, "boom")
}

func TestErrGetBroker(t *testing.T) {
	assertMeshkitError(t, ErrGetBroker(errors.New("boom")), ErrGetBrokerCode, "boom")
}

func TestErrCreateBroker(t *testing.T) {
	assertMeshkitError(t, ErrCreateBroker(errors.New("boom")), ErrCreateBrokerCode, "boom")
}

func TestErrDeleteBroker(t *testing.T) {
	assertMeshkitError(t, ErrDeleteBroker(errors.New("boom")), ErrDeleteBrokerCode, "boom")
}

func TestErrReconcileBroker(t *testing.T) {
	assertMeshkitError(t, ErrReconcileBroker(errors.New("boom")), ErrReconcileBrokerCode, "boom")
}

func TestErrReconcileCR(t *testing.T) {
	assertMeshkitError(t, ErrReconcileCR(errors.New("boom")), ErrReconcileCRCode, "boom")
}

func TestErrCheckHealth(t *testing.T) {
	assertMeshkitError(t, ErrCheckHealth(errors.New("boom")), ErrCheckHealthCode, "boom")
}

func TestErrGetEndpoint(t *testing.T) {
	assertMeshkitError(t, ErrGetEndpoint(errors.New("boom")), ErrGetEndpointCode, "boom")
}

func TestErrUpdateResource(t *testing.T) {
	assertMeshkitError(t, ErrUpdateResource(errors.New("boom")), ErrUpdateResourceCode, "boom")
}

func TestErrMarshal(t *testing.T) {
	assertMeshkitError(t, ErrMarshal(errors.New("boom")), ErrMarshalCode, "boom")
}

// TestErrorCodesUnique guards against the historical code collision between the
// controllers registry and the pkg/* registries by asserting every code in this
// package is distinct.
func TestErrorCodesUnique(t *testing.T) {
	codes := []string{
		ErrGetMeshsyncCode, ErrCreateMeshsyncCode, ErrReconcileMeshsyncCode,
		ErrGetBrokerCode, ErrCreateBrokerCode, ErrReconcileBrokerCode,
		ErrReconcileCRCode, ErrDeleteMeshsyncCode, ErrDeleteBrokerCode,
		ErrCheckHealthCode, ErrGetEndpointCode, ErrUpdateResourceCode, ErrMarshalCode,
	}
	seen := make(map[string]bool, len(codes))
	for _, c := range codes {
		if seen[c] {
			t.Errorf("duplicate error code %q within the controllers registry", c)
		}
		seen[c] = true
	}
}
