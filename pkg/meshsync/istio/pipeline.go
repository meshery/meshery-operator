package istio

import (
	"github.com/myntra/pipeline"
)

var (
	concurrent = true
	sequential = false
)

// this file will create all the stages
// it will just get the steps and we will arrange them here

// New will return a Pipeline
func New() *pipeline.Pipeline {
	// new pipeline
	istioPipeline := pipeline.New("istio-discovery", 1000)

	// creating istio specific stages
	// stage-1
	stage1 := pipeline.NewStage("stage-1", sequential, false)
	// creating steps for this stage
	step1 := &VirtualService{}
	// adding steps to  the  stage
	stage1.AddStep(step1)

	// adding stages to pipeline
	istioPipeline.AddStage(stage1)

	return istioPipeline
}
