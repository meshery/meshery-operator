/*
Copyright 2023 Layer5, Inc.

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
	return errors.New(ErrGetMeshsyncCode + ":" + "Unable to get meshsync resource" + err.Error())
}

func ErrCreateMeshsync(err error) error {
	return errors.New(ErrCreateMeshsyncCode + ":" + "Unable to create meshsync controller" + err.Error())
}

func ErrDeleteMeshsync(err error) error {
	return errors.New(ErrDeleteMeshsyncCode + ":" + "Unable to delete meshsync controller" + err.Error())
}

func ErrReconcileMeshsync(err error) error {
	return errors.New(ErrReconcileMeshsyncCode + ":" + "Error during meshsync resource reconciliation" + err.Error())
}

func ErrGetBroker(err error) error {
	return errors.New(ErrGetBrokerCode + ":" + "Broker resource not found" + err.Error())
}

func ErrCreateBroker(err error) error {
	return errors.New(ErrCreateBrokerCode + ":" + "Unable to create broker controller" + err.Error())
}

func ErrDeleteBroker(err error) error {
	return errors.New(ErrDeleteBrokerCode + ":" + "Unable to delete broker controller" + err.Error())
}

func ErrReconcileBroker(err error) error {
	return errors.New(ErrReconcileBrokerCode + ":" + "Error during broker resource reconciliation" + err.Error())
}

func ErrReconcileCR(err error) error {
	return errors.New(ErrReconcileCRCode + ":" + "Error during custom resource reconciliation" + err.Error())
}

func ErrCheckHealth(err error) error {
	return errors.New(ErrCheckHealthCode + ":" + "Error during health check" + err.Error())
}

func ErrGetEndpoint(err error) error {
	return errors.New(ErrGetEndpointCode + ":" + "Unable to get endpoint" + err.Error())
}

func ErrUpdateResource(err error) error {
	return errors.New(ErrUpdateResourceCode + ":" + "Unable to update resource" + err.Error())
}

func ErrMarshal(err error) error {
	return errors.New(ErrMarshalCode + ":" + "Error during marshaling" + err.Error())
}
