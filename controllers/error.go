package controllers

import (
	"github.com/layer5io/meshkit/errors"
)

const (
	ErrGetMeshsyncCode       = "meshsync_test"
	ErrCreateMeshsyncCode    = "meshsync_test"
	ErrReconcileMeshsyncCode = "meshsync_test"
	ErrGetBrokerCode         = "meshsync_test"
	ErrCreateBrokerCode      = "meshsync_test"
	ErrReconcileBrokerCode   = "meshsync_test"
	ErrReconcileCRCode       = "meshsync_test"
	ErrDeleteMeshsyncCode    = "meshsync_test"
	ErrDeleteBrokerCode      = "meshsync_test"
	ErrCheckHealthCode       = "meshsync_test"
	ErrGetEndpointCode       = "meshsync_test"
	ErrUpdateResourceCode    = "meshsync_test"
)

func ErrGetMeshsync(err error) error {
	return errors.New(ErrGetMeshsyncCode, errors.Alert, []string{"Meshsync resource not found", err.Error()}, []string{}, []string{}, []string{})
}

func ErrCreateMeshsync(err error) error {
	return errors.New(ErrCreateMeshsyncCode, errors.Alert, []string{"Unable to create meshsync controller", err.Error()}, []string{}, []string{}, []string{})
}

func ErrDeleteMeshsync(err error) error {
	return errors.New(ErrDeleteMeshsyncCode, errors.Alert, []string{"Unable to delete meshsync controller", err.Error()}, []string{}, []string{}, []string{})
}

func ErrReconcileMeshsync(err error) error {
	return errors.New(ErrReconcileMeshsyncCode, errors.Alert, []string{"Error during meshsync resource reconcillation", err.Error()}, []string{}, []string{}, []string{})
}

func ErrGetBroker(err error) error {
	return errors.New(ErrGetBrokerCode, errors.Alert, []string{"Broker resource not found", err.Error()}, []string{}, []string{}, []string{})
}

func ErrCreateBroker(err error) error {
	return errors.New(ErrCreateBrokerCode, errors.Alert, []string{"Unable to create broker controller", err.Error()}, []string{}, []string{}, []string{})
}

func ErrDeleteBroker(err error) error {
	return errors.New(ErrDeleteBrokerCode, errors.Alert, []string{"Unable to delete broker controller", err.Error()}, []string{}, []string{}, []string{})
}

func ErrReconcileBroker(err error) error {
	return errors.New(ErrReconcileBrokerCode, errors.Alert, []string{"Error during broker resource reconcillation", err.Error()}, []string{}, []string{}, []string{})
}

func ErrReconcileCR(err error) error {
	return errors.New(ErrReconcileCRCode, errors.Alert, []string{"Error during custom resource resource reconcillation", err.Error()}, []string{}, []string{}, []string{})
}

func ErrCheckHealth(err error) error {
	return errors.New(ErrCheckHealthCode, errors.Alert, []string{"Error during health check", err.Error()}, []string{}, []string{}, []string{})
}

func ErrGetEndpoint(err error) error {
	return errors.New(ErrGetEndpointCode, errors.Alert, []string{"Error gettting endpoint", err.Error()}, []string{}, []string{}, []string{})
}

func ErrUpdateResource(err error) error {
	return errors.New(ErrUpdateResourceCode, errors.Alert, []string{"Error updating Resource", err.Error()}, []string{}, []string{}, []string{})
}
