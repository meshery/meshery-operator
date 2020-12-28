package broker

import (
	"github.com/layer5io/meshkit/errors"
)

const (
	ErrGettingResourceCode  = "meshsync_test"
	ErrReplicasNotReadyCode = "meshsync_test"
	ErrConditionFalseCode   = "meshsync_test"
)

func ErrGettingResource(err error) error {
	return errors.NewDefault(ErrGettingResourceCode, "Unable to get resource", err.Error())
}

func ErrReplicasNotReady(reason string) error {
	return errors.NewDefault(ErrReplicasNotReadyCode, "Replicas not ready", reason)
}

func ErrConditionFalse(reason string) error {
	return errors.NewDefault(ErrConditionFalseCode, reason)
}
