package meshsync

import (
	"context"
	"github.com/layer5io/meshery-operator/pkg/meshsync/models"
	"sync"
)

type Synchronizer interface {
	Synchronize(context.Context, *sync.WaitGroup, <-chan struct{}) error
}

type Fingerprinter interface {
	Fingerprint(context.Context, models.Event)
}

type MeshSync interface {
	Synchronizer
	Fingerprinter
}
