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
)

func ErrGetMeshsync(err error) error {
	return errors.NewDefault(ErrGetMeshsyncCode, "Meshsync resource not found", err.Error())
}

func ErrCreateMeshsync(err error) error {
	return errors.NewDefault(ErrCreateMeshsyncCode, "Unable to create meshsync controller", err.Error())
}

func ErrDeleteMeshsync(err error) error {
	return errors.NewDefault(ErrDeleteMeshsyncCode, "Unable to delete meshsync controller", err.Error())
}

func ErrReconcileMeshsync(err error) error {
	return errors.NewDefault(ErrReconcileMeshsyncCode, "Error during meshsync resource reconcillation", err.Error())
}

func ErrGetBroker(err error) error {
	return errors.NewDefault(ErrGetBrokerCode, "Broker resource not found", err.Error())
}

func ErrCreateBroker(err error) error {
	return errors.NewDefault(ErrCreateBrokerCode, "Unable to create broker controller", err.Error())
}

func ErrDeleteBroker(err error) error {
	return errors.NewDefault(ErrDeleteBrokerCode, "Unable to delete broker controller", err.Error())
}

func ErrReconcileBroker(err error) error {
	return errors.NewDefault(ErrReconcileBrokerCode, "Error during broker resource reconcillation", err.Error())
}

func ErrReconcileCR(err error) error {
	return errors.NewDefault(ErrReconcileCRCode, "Error during custom resource reconcillation", err.Error())
}

func ErrCheckHealth(err error) error {
	return errors.NewDefault(ErrCheckHealthCode, "Error during health check", err.Error())
}

func ErrGetEndpoint(err error) error {
	return errors.NewDefault(ErrGetEndpointCode, "Error getting endpoint", err.Error())
}
