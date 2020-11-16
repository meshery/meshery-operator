package pipeline

import (
	"github.com/myntra/pipeline"
)

type Options struct {
	Pipeline *pipeline.Pipeline `json:"pipeline,omitempty"`
	SkipFail bool               `json:"skipfail,omitempty"` // This flag determnines whether to skip this pipeline during failure or to abort the program
}

// NewPipeline will return a new Pipeline handler
func New(opts Options) (*pipeline.Pipeline, error) {
	pl := pipeline.New(opts.Pipeline.Name, 1000)
	for _, pipeStage := range opts.Pipeline.Stages {
		stage := pipeline.NewStage(pipeStage.Name, pipeStage.Concurrent, false)
		for _, pipeStep := range pipeStage.Steps {
			pipeStage.AddStep(pipeStep)
		}
		pl.AddStage(stage)
	}
	return pl, nil
}
