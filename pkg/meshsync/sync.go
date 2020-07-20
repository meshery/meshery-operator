package meshsync

import (
	"context"
	"sync"
)

type Synchronizer interface {
	Synchronize(context.Context, *sync.WaitGroup, <-chan struct{}) error
	IsDeployed() bool
}
