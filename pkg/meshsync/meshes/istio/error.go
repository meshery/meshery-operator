package istio

import (
	"github.com/layer5io/meshkit/errors"
)

const (
	ErrInitPipelineCode = "11000"
	ErrInitInformerCode = "11001"
)

func ErrInitPipeline(err interface{}) error {
	return errors.NewDefault(ErrInitPipelineCode, "Pipelines failed", err.(error).Error())
}

func ErrInitInformer(err error) error {
	return errors.NewDefault(ErrInitInformerCode, "Informers failed", err.Error())
}
