/*
Copyright 2020 Layer5, Inc.

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
	"github.com/layer5io/meshkit/errors"
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
	ErrUpdateMeshsyncCode    = "1013"
)

// Error definitions
func ErrCreateMeshsync(err error) error {
	return errors.New(ErrCreateMeshsyncCode, errors.Alert, []string{"Unable to create meshsync controller"}, []string{err.Error()}, []string{}, []string{})
}

func ErrGetMeshsync(err error) error {
	return errors.New(ErrGetMeshsyncCode, errors.Alert, []string{"Meshsync resource not found"}, []string{err.Error()}, []string{}, []string{})
}

func ErrUpdateMeshsync(err error) error {
	return errors.New(ErrUpdateMeshsyncCode, errors.Alert, []string{"Unable to update meshsync controller"}, []string{err.Error()}, []string{}, []string{})
}

func ErrDeleteMeshsync(err error) error {
	return errors.New(ErrDeleteMeshsyncCode, errors.Alert, []string{"Unable to delete meshsync controller"}, []string{err.Error()}, []string{}, []string{})
}

func ErrReconcileMeshsync(err error) error {
	return errors.New(ErrReconcileMeshsyncCode, errors.Alert, []string{"Error during meshsync resource reconciliation"}, []string{err.Error()}, []string{}, []string{})
}

func ErrGetBroker(err error) error {
	return errors.New(ErrGetBrokerCode, errors.Alert, []string{"Broker resource not found"}, []string{err.Error()}, []string{}, []string{})
}

func ErrCreateBroker(err error) error {
	return errors.New(ErrCreateBrokerCode, errors.Alert, []string{"Unable to create broker controller"}, []string{err.Error()}, []string{}, []string{})
}

func ErrDeleteBroker(err error) error {
	return errors.New(ErrDeleteBrokerCode, errors.Alert, []string{"Unable to delete broker controller"}, []string{err.Error()}, []string{}, []string{})
}

func ErrReconcileBroker(err error) error {
	return errors.New(ErrReconcileBrokerCode, errors.Alert, []string{"Error during broker resource reconciliation"}, []string{err.Error()}, []string{}, []string{})
}

func ErrReconcileCR(err error) error {
	return errors.New(ErrReconcileCRCode, errors.Alert, []string{"Error during custom resource reconciliation"}, []string{err.Error()}, []string{}, []string{})
}

func ErrCheckHealth(err error) error {
	return errors.New(ErrCheckHealthCode, errors.Alert, []string{"Error during health check"}, []string{err.Error()}, []string{}, []string{})
}

func ErrGetEndpoint(err error) error {
	return errors.New(ErrGetEndpointCode, errors.Alert, []string{"Error getting endpoint"}, []string{err.Error()}, []string{}, []string{})
}

func ErrUpdateResource(err error) error {
	return errors.New(ErrUpdateResourceCode, errors.Alert, []string{"Error updating resource"}, []string{err.Error()}, []string{}, []string{})
}
