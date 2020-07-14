package controller

import (
	"meshery-operator/pkg/meshsync"
	"sync"
)

type controller struct {
	syncs []meshsync.Synchronizer
}

func New(s ...meshsync.Synchronizer) (*controller, error) {
	return &controller{
		syncs: s,
	}, nil
}

func (ctrl *controller) Run(quit <-chan struct{}) error {
	wg := &sync.WaitGroup{}
	for _, sync := range ctrl.syncs {
		wg.Add(1)
		go sync.Synchronize(wg, quit)
	}

	wg.Wait()
	return nil
}
