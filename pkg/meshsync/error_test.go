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

func TestErrGettingMeshsyncResource(t *testing.T) {
	assertMeshkitError(t, ErrGettingMeshsyncResource(errors.New("boom")), ErrGettingMeshsyncResourceCode, "boom")
}

func TestErrGettingMeshsyncEndpoint(t *testing.T) {
	assertMeshkitError(t, ErrGettingMeshsyncEndpoint(errors.New("boom")), ErrGettingMeshsyncEndpointCode, "boom")
}

func TestErrMeshsyncReplicasNotReady(t *testing.T) {
	assertMeshkitError(t, ErrMeshsyncReplicasNotReady("not enough replicas"), ErrMeshsyncReplicasNotReadyCode, "not enough replicas")
}

func TestErrMeshsyncConditionFalse(t *testing.T) {
	assertMeshkitError(t, ErrMeshsyncConditionFalse("condition false"), ErrMeshsyncConditionFalseCode, "condition false")
}

// TestErrorCodesDistinctFromBroker documents that the meshsync registry was
// renumbered (1018-1021) to no longer collide with the broker registry
// (1013-1016).
func TestErrorCodesDistinctFromBroker(t *testing.T) {
	brokerCodes := map[string]bool{"1013": true, "1014": true, "1015": true, "1016": true}
	for _, c := range []string{ErrGettingMeshsyncResourceCode, ErrMeshsyncReplicasNotReadyCode, ErrMeshsyncConditionFalseCode, ErrGettingMeshsyncEndpointCode} {
		if brokerCodes[c] {
			t.Errorf("meshsync error code %q collides with the broker registry", c)
		}
	}
}
