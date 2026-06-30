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
	meshkiterrors "github.com/meshery/meshkit/errors"
)

// Error codes. Names and codes are unique within the meshery-operator component;
// allocate the next free code from helpers/component_info.json and bump its
// next_error_code. These were renumbered from 1013-1016 and renamed to resolve a
// code-and-name collision with the pkg/broker registry.
const (
	ErrGettingMeshsyncResourceCode  = "1018"
	ErrMeshsyncReplicasNotReadyCode = "1019"
	ErrMeshsyncConditionFalseCode   = "1020"
	ErrGettingMeshsyncEndpointCode  = "1021"
)

// ErrGettingMeshsyncResource is returned when a MeshSync-owned resource cannot be fetched.
func ErrGettingMeshsyncResource(err error) error {
	return meshkiterrors.New(ErrGettingMeshsyncResourceCode, meshkiterrors.Alert,
		[]string{"Unable to get a MeshSync-owned resource"},
		[]string{err.Error()},
		[]string{"The MeshSync Deployment lookup failed"},
		[]string{"Check the operator RBAC and confirm the Deployment exists in the MeshSync namespace"})
}

// ErrGettingMeshsyncEndpoint is returned when the MeshSync broker endpoint cannot be resolved.
func ErrGettingMeshsyncEndpoint(err error) error {
	return meshkiterrors.New(ErrGettingMeshsyncEndpointCode, meshkiterrors.Alert,
		[]string{"Unable to get the MeshSync broker endpoint"},
		[]string{err.Error()},
		[]string{"The broker referenced by the MeshSync spec has no resolvable endpoint"},
		[]string{"Verify the native broker CR or the custom broker URL configured in the MeshSync spec"})
}

// ErrMeshsyncReplicasNotReady is returned when the MeshSync workload has not
// reached its desired ready replica count.
func ErrMeshsyncReplicasNotReady(reason string) error {
	return meshkiterrors.New(ErrMeshsyncReplicasNotReadyCode, meshkiterrors.Alert,
		[]string{"MeshSync replicas are not ready"},
		[]string{reason},
		[]string{"The MeshSync Deployment has not reached its desired ReadyReplicas"},
		[]string{"Inspect the Deployment pod status and events; the controller will requeue and retry"})
}

// ErrMeshsyncConditionFalse is returned when a required MeshSync readiness condition reports false.
func ErrMeshsyncConditionFalse(reason string) error {
	return meshkiterrors.New(ErrMeshsyncConditionFalseCode, meshkiterrors.Alert,
		[]string{"A MeshSync readiness condition is false"},
		[]string{reason},
		[]string{"A required status condition on the MeshSync workload reports False"},
		[]string{"Inspect the MeshSync status conditions for the reported reason"})
}
