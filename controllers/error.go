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
	meshkiterrors "github.com/meshery/meshkit/errors"
)

// Error codes. Codes are unique within this component; allocate the next free
// code from helpers/component_info.json and bump its next_error_code.
const (
	ErrGetMeshsyncCode       = "1001"
	ErrCreateMeshsyncCode    = "1002"
	ErrReconcileMeshsyncCode = "1003"
	ErrGetBrokerCode         = "1004"
	ErrCreateBrokerCode      = "1005"
	ErrReconcileBrokerCode   = "1006"
	ErrReconcileCRCode       = "1007"
	ErrDeleteMeshsyncCode    = "1008"
	ErrDeleteBrokerCode      = "1009"
	ErrCheckHealthCode       = "1010"
	ErrGetEndpointCode       = "1011"
	ErrUpdateResourceCode    = "1012"
	ErrMarshalCode           = "1017"
)

// ErrGetMeshsync is returned when the MeshSync custom resource cannot be fetched.
func ErrGetMeshsync(err error) error {
	return meshkiterrors.New(ErrGetMeshsyncCode, meshkiterrors.Alert,
		[]string{"Unable to get the MeshSync resource"},
		[]string{err.Error()},
		[]string{"The MeshSync custom resource was deleted, or operator RBAC denies get/list/watch on meshsyncs"},
		[]string{"Confirm the MeshSync CR exists and that the operator's ClusterRole grants get/list/watch on meshsyncs.meshery.io"})
}

// ErrCreateMeshsync is returned when the MeshSync workload cannot be created.
func ErrCreateMeshsync(err error) error {
	return meshkiterrors.New(ErrCreateMeshsyncCode, meshkiterrors.Alert,
		[]string{"Unable to create the MeshSync workload"},
		[]string{err.Error()},
		[]string{"The MeshSync Deployment or its owned objects could not be created"},
		[]string{"Check the operator RBAC for deployments and inspect the events in the MeshSync namespace"})
}

// ErrDeleteMeshsync is returned when MeshSync-owned objects cannot be deleted.
func ErrDeleteMeshsync(err error) error {
	return meshkiterrors.New(ErrDeleteMeshsyncCode, meshkiterrors.Alert,
		[]string{"Unable to delete the MeshSync workload"},
		[]string{err.Error()},
		[]string{"Owned MeshSync objects could not be deleted during finalization"},
		[]string{"Inspect the MeshSync finalizers and confirm the operator RBAC grants delete on deployments"})
}

// ErrReconcileMeshsync is returned when MeshSync reconciliation fails.
func ErrReconcileMeshsync(err error) error {
	return meshkiterrors.New(ErrReconcileMeshsyncCode, meshkiterrors.Alert,
		[]string{"MeshSync reconciliation failed"},
		[]string{err.Error()},
		[]string{"Broker configuration, workload deployment, or the health check failed for MeshSync"},
		[]string{"Check the MeshSync status conditions and the operator logs for the failing reconcile step"})
}

// ErrGetBroker is returned when the Broker custom resource cannot be fetched.
func ErrGetBroker(err error) error {
	return meshkiterrors.New(ErrGetBrokerCode, meshkiterrors.Alert,
		[]string{"Unable to get the Broker resource"},
		[]string{err.Error()},
		[]string{"The Broker custom resource was deleted, or operator RBAC denies get/list/watch on brokers"},
		[]string{"Confirm the Broker CR exists and that the operator's ClusterRole grants get/list/watch on brokers.meshery.io"})
}

// ErrCreateBroker is returned when the Broker workload cannot be created.
func ErrCreateBroker(err error) error {
	return meshkiterrors.New(ErrCreateBrokerCode, meshkiterrors.Alert,
		[]string{"Unable to create the Broker workload"},
		[]string{err.Error()},
		[]string{"The NATS StatefulSet, Service, or ConfigMaps could not be created"},
		[]string{"Check the operator RBAC and inspect the events on the owned objects in the broker namespace"})
}

// ErrDeleteBroker is returned when Broker-owned objects cannot be deleted.
func ErrDeleteBroker(err error) error {
	return meshkiterrors.New(ErrDeleteBrokerCode, meshkiterrors.Alert,
		[]string{"Unable to delete the Broker workload"},
		[]string{err.Error()},
		[]string{"Owned broker objects could not be deleted during finalization"},
		[]string{"Inspect the Broker finalizers and confirm the operator RBAC grants delete on statefulsets, services, and configmaps"})
}

// ErrReconcileBroker is returned when Broker reconciliation fails.
func ErrReconcileBroker(err error) error {
	return meshkiterrors.New(ErrReconcileBrokerCode, meshkiterrors.Alert,
		[]string{"Broker reconciliation failed"},
		[]string{err.Error()},
		[]string{"The NATS StatefulSet, Service, or ConfigMaps could not be created or updated"},
		[]string{"Check the operator RBAC and inspect the events on the owned objects in the broker namespace"})
}

// ErrReconcileCR is returned when a custom resource reconcile step fails.
func ErrReconcileCR(err error) error {
	return meshkiterrors.New(ErrReconcileCRCode, meshkiterrors.Alert,
		[]string{"Custom resource reconciliation failed"},
		[]string{err.Error()},
		[]string{"A reconcile step failed for the custom resource"},
		[]string{"Check the operator logs for the underlying cause; the controller will requeue and retry"})
}

// ErrCheckHealth is returned when a managed workload fails its health check.
func ErrCheckHealth(err error) error {
	return meshkiterrors.New(ErrCheckHealthCode, meshkiterrors.Alert,
		[]string{"Workload health check failed"},
		[]string{err.Error()},
		[]string{"The StatefulSet or Deployment has not reached its desired ReadyReplicas"},
		[]string{"Inspect the pod status and events for the workload; the controller will requeue and retry"})
}

// ErrGetEndpoint is returned when the broker endpoint cannot be derived.
func ErrGetEndpoint(err error) error {
	return meshkiterrors.New(ErrGetEndpointCode, meshkiterrors.Alert,
		[]string{"Unable to derive the broker endpoint"},
		[]string{err.Error()},
		[]string{"The broker Service has no reachable internal or external address yet"},
		[]string{"Verify the broker Service exists and has assigned ports and an address; LoadBalancer addresses may still be pending"})
}

// ErrUpdateResource is returned when a resource or its status cannot be updated.
func ErrUpdateResource(err error) error {
	return meshkiterrors.New(ErrUpdateResourceCode, meshkiterrors.Alert,
		[]string{"Unable to update the resource"},
		[]string{err.Error()},
		[]string{"A status patch or resource update conflicted or was rejected by the API server"},
		[]string{"Check for concurrent writers and confirm the operator RBAC grants update on the resource and its status; the controller will retry"})
}

// ErrMarshal is returned when a resource cannot be marshaled for a status patch.
func ErrMarshal(err error) error {
	return meshkiterrors.New(ErrMarshalCode, meshkiterrors.Alert,
		[]string{"Unable to marshal the resource"},
		[]string{err.Error()},
		[]string{"The custom resource could not be serialized for a status patch"},
		[]string{"This indicates a defect in the status builder; please report it with the operator logs"})
}
