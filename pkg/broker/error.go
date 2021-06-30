package broker

import (
	"github.com/layer5io/meshkit/errors"
)

const (
	ErrGettingResourceCode  = "1013"
	ErrReplicasNotReadyCode = "1014"
	ErrConditionFalseCode   = "1015"
	ErrGettingEndpointCode  = "1016"
)

func ErrGettingResource(err error) error {
	return errors.New(ErrGettingResourceCode, errors.Alert, []string{"Unable to get requested resource"}, []string{err.Error()}, []string{}, []string{})
}

func ErrGettingEndpoint(err error) error {
	return errors.New(ErrGettingEndpointCode, errors.Alert, []string{"Unable to discovery endpoint"}, []string{err.Error()}, []string{}, []string{})
}

func ErrReplicasNotReady(reason string) error {
	return errors.New(ErrReplicasNotReadyCode, errors.Alert, []string{"Replicas not ready"}, []string{reason}, []string{}, []string{})
}

func ErrConditionFalse(reason string) error {
	return errors.New(ErrConditionFalseCode, errors.Alert, []string{reason}, []string{}, []string{}, []string{})
}
