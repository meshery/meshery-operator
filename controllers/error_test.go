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
	"testing"
)

func TestErrGetMeshsync(t *testing.T) {
	err := errors.New("test error")
	if ErrGetMeshsync(err).Error() != "1001:Unable to get meshsync resource" {
		t.Error("ErrGetMeshsync error")
	}
}
func TestErrCreateMeshsync(t *testing.T) {
	err := errors.New("test error")
	if ErrCreateMeshsync(err).Error() != "1002:Unable to create meshsync controller" {
		t.Error("ErrCreateMeshsync error")
	}
}

func TestErrDeleteMeshsync(t *testing.T) {
	err := errors.New("test error")
	if ErrDeleteMeshsync(err).Error() != "1008:Unable to delete meshsync controller" {
		t.Error("ErrDeleteMeshsync error")
	}
}

func TestErrReconcileMeshsync(t *testing.T) {
	err := errors.New("test error")
	if ErrReconcileMeshsync(err).Error() != "1003:Error during meshsync resource reconciliation" {
		t.Error("ErrReconcileMeshsync error")
	}
}

// Test case for ErrGetBroker
func TestErrGetBroker(t *testing.T) {
	err := errors.New("test error")
	if ErrGetBroker(err).Error() != "1004:Broker resource not found" {
		t.Error("ErrGetBroker error")
	}
}

// Test case for ErrCreateBroker
func TestErrCreateBroker(t *testing.T) {
	err := errors.New("test error")
	if ErrCreateBroker(err).Error() != "1005:Unable to create broker controller" {
		t.Error("ErrCreateBroker error")
	}
}

// Test case for ErrDeleteBroker
func TestErrDeleteBroker(t *testing.T) {
	err := errors.New("test error")
	if ErrDeleteBroker(err).Error() != "1009:Unable to delete broker controller" {
		t.Error("ErrDeleteBroker error")
	}
}

// Test case for ErrReconcileBroker
func TestErrReconcileBroker(t *testing.T) {
	err := errors.New("test error")
	if ErrReconcileBroker(err).Error() != "1006:Error during broker resource reconciliation" {
		t.Error("ErrReconcileBroker error")
	}
}

// Test case for ErrReconcileCR
func TestErrReconcileCR(t *testing.T) {
	err := errors.New("test error")
	if ErrReconcileCR(err).Error() != "1007:Error during custom resource reconciliation" {
		t.Error("ErrReconcileCR error")
	}
}

// Test case for ErrCheckHealth
func TestErrCheckHealth(t *testing.T) {
	err := errors.New("test error")
	if ErrCheckHealth(err).Error() != "1010:Error during health check" {
		t.Error("ErrCheckHealth error")
	}
}

// Test case for ErrGetEndpoint
func TestErrGetEndpoint(t *testing.T) {
	err := errors.New("test error")
	if ErrGetEndpoint(err).Error() != "1011:Unable to get endpoint" {
		t.Error("ErrGetEndpoint error")
	}
}

// Test case for ErrUpdateResource
func TestErrUpdateResource(t *testing.T) {
	err := errors.New("test error")
	if ErrUpdateResource(err).Error() != "1012:Unable to update resource" {
		t.Error("ErrUpdateResource error")
	}
}

// Test case for ErrMarshal
func TestErrMarshal(t *testing.T) {
	err := errors.New("test error")
	if ErrMarshal(err).Error() != "11049:Error during marshaling" {
		t.Error("ErrMarshal error")
	}
}
