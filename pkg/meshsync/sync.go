package meshsync

import (
	"context"
	"meshery-operator/pkg/meshsync/models"
	"sync"
)

type Synchronizer interface {
	Synchronize(context.Context, *sync.WaitGroup, <-chan struct{}) error
}

type Processor interface {
	Process(context.Context, models.Event)
}

type MeshSync interface {
	Synchronizer
	Processor
}
