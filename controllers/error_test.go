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
)

func TestErrGetMeshsync(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrGetMeshsync(testErr)

	expectedPrefix := "1001: Unable to get meshsync resource:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	// Test error unwrapping
	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

func TestErrCreateMeshsync(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrCreateMeshsync(testErr)

	expectedPrefix := "1002: Unable to create meshsync controller:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

func TestErrDeleteMeshsync(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrDeleteMeshsync(testErr)

	expectedPrefix := "1008: Unable to delete meshsync controller:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

func TestErrReconcileMeshsync(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrReconcileMeshsync(testErr)

	expectedPrefix := "1003: Error during meshsync resource reconciliation:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}
}

// Test case for ErrGetBroker
func TestErrGetBroker(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrGetBroker(testErr)

	expectedPrefix := "1004: Broker resource not found:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

// Test case for ErrCreateBroker
func TestErrCreateBroker(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrCreateBroker(testErr)

	expectedPrefix := "1005: Unable to create broker controller:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

// Test case for ErrDeleteBroker
func TestErrDeleteBroker(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrDeleteBroker(testErr)

	expectedPrefix := "1009: Unable to delete broker controller:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

// Test case for ErrReconcileBroker
func TestErrReconcileBroker(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrReconcileBroker(testErr)

	expectedPrefix := "1006: Error during broker resource reconciliation:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

// Test case for ErrReconcileCR
func TestErrReconcileCR(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrReconcileCR(testErr)

	expectedPrefix := "1007: Error during custom resource reconciliation:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

// Test case for ErrCheckHealth
func TestErrCheckHealth(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrCheckHealth(testErr)

	expectedPrefix := "1010: Error during health check:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

// Test case for ErrGetEndpoint
func TestErrGetEndpoint(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrGetEndpoint(testErr)

	expectedPrefix := "1011: Unable to get endpoint:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

// Test case for ErrUpdateResource
func TestErrUpdateResource(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrUpdateResource(testErr)

	expectedPrefix := "1012: Unable to update resource:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}

// Test case for ErrMarshal
func TestErrMarshal(t *testing.T) {
	testErr := errors.New("test error")
	err := ErrMarshal(testErr)

	expectedPrefix := "11049: Error during marshaling:"
	if !strings.Contains(err.Error(), expectedPrefix) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedPrefix, err.Error())
	}

	if !errors.Is(err, testErr) {
		t.Error("Expected error to wrap the original error")
	}
}
