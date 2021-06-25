package broker

import (
	"github.com/layer5io/meshkit/errors"
)

const (
	ErrGettingResourceCode  = "meshsync_test"
	ErrReplicasNotReadyCode = "meshsync_test"
	ErrConditionFalseCode   = "meshsync_test"
	ErrGettingEndpointCode  = "meshsync_test"
)

func ErrGettingResource(err error) error {
	return errors.New(ErrGettingResourceCode, errors.Alert, []string{"Unable to get requested resource" + err.Error()}, []string{"Unable to get requested resource while doing health check", err.Error()}, []string{}, []string{})
}

func ErrGettingEndpoint(err error) error {
	return errors.New(ErrGettingEndpointCode, errors.Alert, []string{"Unable to discovery endpoint" + err.Error()}, []string{}, []string{}, []string{})
}

func ErrReplicasNotReady(reason string) error {
	return errors.New(ErrReplicasNotReadyCode, errors.Alert, []string{"Replicas not ready.", reason}, []string{}, []string{}, []string{})
}

func ErrConditionFalse(reason string) error {
	return errors.New(ErrConditionFalseCode, errors.Alert, []string{"Health check condition false.", reason}, []string{}, []string{}, []string{})
}
