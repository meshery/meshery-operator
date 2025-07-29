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
	"fmt"
)

// Error codes
// @Aisuko Is there any way to make this more flexible?
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
	ErrMarshalCode           = "11049"
)

// Error definitions
func ErrGetMeshsync(err error) error {
	return fmt.Errorf("%s: Unable to get meshsync resource: %w", ErrGetMeshsyncCode, err)
}

func ErrCreateMeshsync(err error) error {
	return fmt.Errorf("%s: Unable to create meshsync controller: %w", ErrCreateMeshsyncCode, err)
}

func ErrDeleteMeshsync(err error) error {
	return fmt.Errorf("%s: Unable to delete meshsync controller: %w", ErrDeleteMeshsyncCode, err)
}

func ErrReconcileMeshsync(err error) error {
	return fmt.Errorf("%s: Error during meshsync resource reconciliation: %w", ErrReconcileMeshsyncCode, err)
}

func ErrGetBroker(err error) error {
	return fmt.Errorf("%s: Broker resource not found: %w", ErrGetBrokerCode, err)
}

func ErrCreateBroker(err error) error {
	return fmt.Errorf("%s: Unable to create broker controller: %w", ErrCreateBrokerCode, err)
}

func ErrDeleteBroker(err error) error {
	return fmt.Errorf("%s: Unable to delete broker controller: %w", ErrDeleteBrokerCode, err)
}

func ErrReconcileBroker(err error) error {
	return fmt.Errorf("%s: Error during broker resource reconciliation: %w", ErrReconcileBrokerCode, err)
}

func ErrReconcileCR(err error) error {
	return fmt.Errorf("%s: Error during custom resource reconciliation: %w", ErrReconcileCRCode, err)
}

func ErrCheckHealth(err error) error {
	return fmt.Errorf("%s: Error during health check: %w", ErrCheckHealthCode, err)
}

func ErrGetEndpoint(err error) error {
	return fmt.Errorf("%s: Unable to get endpoint: %w", ErrGetEndpointCode, err)
}

func ErrUpdateResource(err error) error {
	return fmt.Errorf("%s: Unable to update resource: %w", ErrUpdateResourceCode, err)
}

func ErrMarshal(err error) error {
	return fmt.Errorf("%s: Error during marshaling: %w", ErrMarshalCode, err)
}
