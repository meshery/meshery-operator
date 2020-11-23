package controllers

import (
	"github.com/layer5io/meshkit/errors"
)

const (
	ErrGetMeshsyncCode    = "meshsync_test"
	ErrCreateMeshsyncCode = "meshsync_test"
)

func ErrGetMeshsync(err error) error {
	return errors.NewDefault(ErrGetMeshsyncCode, "Meshsync resource not found: ", err.Error())
}

func ErrCreateMeshsync(err error) error {
	return errors.NewDefault(ErrCreateMeshsyncCode, "Unable to create meshsync controller: ", err.Error())
}
