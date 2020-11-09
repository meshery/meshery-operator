package istio

import (
	"log"

	"github.com/myntra/pipeline"
)

// VirtualService will implement step interface for VirtualService
type VirtualService struct {
	pipeline.StepContext
}

// Exec - step interface
func (vs *VirtualService) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("Virtual Service Discovery Started")

	return &pipeline.Result{
		Error:  nil,
		Data:   struct{ msg string }{},
		KeyVal: map[string]interface{}{},
	}
}

// Cancel - step interface
func (vs *VirtualService) Cancel() error {
	vs.Status("cancel step")
	return nil
}
