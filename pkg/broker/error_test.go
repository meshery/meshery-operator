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

package broker

import (
	"errors"
	"strings"
	"testing"

	meshkiterrors "github.com/meshery/meshkit/errors"
)

func assertMeshkitError(t *testing.T, err error, wantCode, wantDetail string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := meshkiterrors.GetCode(err); got != wantCode {
		t.Errorf("expected code %q, got %q", wantCode, got)
	}
	if got := meshkiterrors.GetSeverity(err); got != meshkiterrors.Alert {
		t.Errorf("expected severity Alert, got %d", got)
	}
	if wantDetail != "" && !strings.Contains(err.Error(), wantDetail) {
		t.Errorf("expected error to carry detail %q, got: %s", wantDetail, err.Error())
	}
}

func TestErrGettingBrokerResource(t *testing.T) {
	assertMeshkitError(t, ErrGettingBrokerResource(errors.New("boom")), ErrGettingBrokerResourceCode, "boom")
}

func TestErrGettingBrokerEndpoint(t *testing.T) {
	assertMeshkitError(t, ErrGettingBrokerEndpoint(errors.New("boom")), ErrGettingBrokerEndpointCode, "boom")
}

func TestErrBrokerReplicasNotReady(t *testing.T) {
	assertMeshkitError(t, ErrBrokerReplicasNotReady("not enough replicas"), ErrBrokerReplicasNotReadyCode, "not enough replicas")
}

func TestErrBrokerConditionFalse(t *testing.T) {
	assertMeshkitError(t, ErrBrokerConditionFalse("condition false"), ErrBrokerConditionFalseCode, "condition false")
}

// TestErrorCodesUnique guards against re-introducing the historical code
// collision; every code in this package must be distinct.
func TestErrorCodesUnique(t *testing.T) {
	codes := []string{ErrGettingBrokerResourceCode, ErrBrokerReplicasNotReadyCode, ErrBrokerConditionFalseCode, ErrGettingBrokerEndpointCode}
	seen := make(map[string]bool, len(codes))
	for _, c := range codes {
		if seen[c] {
			t.Errorf("duplicate error code %q within the broker registry", c)
		}
		seen[c] = true
	}
}
