package service

import (
	"github.com/layer5io/meshkit/errors"
)

const (
	ErrPanicCode        = "11000"
	ErrGrpcListenerCode = "11001"
	ErrGrpcServerCode   = "11002"
)

func ErrPanic(err interface{}) error {
	return errors.NewDefault(ErrPanicCode, "Program panic error", err.(error).Error())
}

func ErrGrpcListener(err error) error {
	return errors.NewDefault(ErrGrpcListenerCode, "Error creating GRPC listener", err.Error())
}

func ErrGrpcServer(err error) error {
	return errors.NewDefault(ErrGrpcServerCode, "Error creating GRPC server", err.Error())
}
