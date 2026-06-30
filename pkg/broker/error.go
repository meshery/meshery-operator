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
	meshkiterrors "github.com/meshery/meshkit/errors"
)

// Error codes. Names and codes are unique within the meshery-operator component;
// allocate the next free code from helpers/component_info.json and bump its
// next_error_code.
const (
	ErrGettingBrokerResourceCode  = "1013"
	ErrBrokerReplicasNotReadyCode = "1014"
	ErrBrokerConditionFalseCode   = "1015"
	ErrGettingBrokerEndpointCode  = "1016"
)

// ErrGettingBrokerResource is returned when a broker-owned resource cannot be fetched.
func ErrGettingBrokerResource(err error) error {
	return meshkiterrors.New(ErrGettingBrokerResourceCode, meshkiterrors.Alert,
		[]string{"Unable to get a broker-owned resource"},
		[]string{err.Error()},
		[]string{"The broker StatefulSet, Service, or ConfigMap lookup failed"},
		[]string{"Check the operator RBAC and confirm the resource exists in the broker namespace"})
}

// ErrGettingBrokerEndpoint is returned when the broker endpoint cannot be resolved.
func ErrGettingBrokerEndpoint(err error) error {
	return meshkiterrors.New(ErrGettingBrokerEndpointCode, meshkiterrors.Alert,
		[]string{"Unable to get the broker endpoint"},
		[]string{err.Error()},
		[]string{"The broker Service is not yet reachable or has no assigned address"},
		[]string{"Verify the broker Service and its ports; a pending LoadBalancer address resolves on a later reconcile"})
}

// ErrBrokerReplicasNotReady is returned when the broker workload has not reached
// its desired ready replica count.
func ErrBrokerReplicasNotReady(reason string) error {
	return meshkiterrors.New(ErrBrokerReplicasNotReadyCode, meshkiterrors.Alert,
		[]string{"Broker replicas are not ready"},
		[]string{reason},
		[]string{"The NATS StatefulSet has not reached its desired ReadyReplicas"},
		[]string{"Inspect the StatefulSet pod status and events; the controller will requeue and retry"})
}

// ErrBrokerConditionFalse is returned when a required broker readiness condition reports false.
func ErrBrokerConditionFalse(reason string) error {
	return meshkiterrors.New(ErrBrokerConditionFalseCode, meshkiterrors.Alert,
		[]string{"A broker readiness condition is false"},
		[]string{reason},
		[]string{"A required status condition on the broker workload reports False"},
		[]string{"Inspect the broker status conditions for the reported reason"})
}
