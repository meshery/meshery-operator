package service

import (
	"github.com/layer5io/meshkit/errors"
)

const (
	ErrPanicCode        = "11000"
	ErrGrpcListenerCode = "11001"
	ErrGrpcServerCode   = "11002"
	ErrNewDiscoveryCode = "11003"
	ErrNewInformerCode  = "11004"
	ErrSetupClusterCode = "11005"
	ErrSetupIstioCode   = "11006"
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

func ErrNewDiscovery(err error) error {
	return errors.NewDefault(ErrNewDiscoveryCode, "Error creating discovery client", err.Error())
}

func ErrNewInformer(err error) error {
	return errors.NewDefault(ErrNewInformerCode, "Error creating informer client", err.Error())
}

func ErrSetupCluster(err error) error {
	return errors.NewDefault(ErrSetupClusterCode, "Error setting up cluster flow", err.Error())
}

func ErrSetupIstio(err error) error {
	return errors.NewDefault(ErrSetupIstioCode, "Error setting up istio flow", err.Error())
}
