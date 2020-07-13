package meshsync

import "sync"

type Synchronizer interface {
	Synchronize(*sync.WaitGroup, <-chan struct{}) error
	IsDeployed() bool
}
