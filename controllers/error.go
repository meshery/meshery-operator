package controllers

import (
	"github.com/layer5io/meshkit/errors"
)

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
)

func ErrGetMeshsync(err error) error {
	return errors.New(ErrGetMeshsyncCode, errors.Alert, []string{"Meshsync resource not found"}, []string{err.Error()}, []string{}, []string{})
}

func ErrCreateMeshsync(err error) error {
	return errors.New(ErrCreateMeshsyncCode, errors.Alert, []string{"Unable to create meshsync controller"}, []string{err.Error()}, []string{}, []string{})
}

func ErrDeleteMeshsync(err error) error {
	return errors.New(ErrDeleteMeshsyncCode, errors.Alert, []string{"Unable to delete meshsync controller"}, []string{err.Error()}, []string{}, []string{})
}

func ErrReconcileMeshsync(err error) error {
	return errors.New(ErrReconcileMeshsyncCode, errors.Alert, []string{"Error during meshsync resource reconcillation"}, []string{err.Error()}, []string{}, []string{})
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
	return errors.New(ErrReconcileBrokerCode, errors.Alert, []string{"Error during broker resource reconcillation"}, []string{err.Error()}, []string{}, []string{})
}

func ErrReconcileCR(err error) error {
	return errors.New(ErrReconcileCRCode, errors.Alert, []string{"Error during custom resource reconcillation"}, []string{err.Error()}, []string{}, []string{})
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
