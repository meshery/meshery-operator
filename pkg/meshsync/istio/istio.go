package istio

import (
	"context"
	"meshery-operator/pkg/kube"
	"meshery-operator/pkg/meshsync/models"
	"sync"
	"time"

	"log"

	ikube "istio.io/istio/pkg/kube"
)

// Istio implements the Synchronizer interface for Istio ServiceMesh
type Istio struct {
	// namespace of the istio controlplane
	namespace string

	// cli is istio's extended client
	cli ikube.ExtendedClient
	// kcli is a instance for the kubernetes client helper
	kcli *kube.Client

	// evCh receives all events from the synchronizer
	evCh chan models.Event
	// errCh receives all errors from the synchronizer
	errCh chan error
}

// New returns a new instance of the Istio synchronizer
func New(kcli *kube.Client, ns string) (*Istio, error) {
	extendedClient, err := ikube.NewExtendedClient(ikube.BuildClientCmd(kcli.Kubeconfig(), ""), "")
	if err != nil {
		return nil, err
	}
	return &Istio{
		cli:       extendedClient,
		namespace: ns,
		kcli:      kcli,
		evCh:      make(chan models.Event, 1),
		errCh:     make(chan error, 1),
	}, nil
}

// Synchronize periodically queries the Service Mesh control-plane
func (i *Istio) Synchronize(ctx context.Context, wg *sync.WaitGroup, quit <-chan struct{}) error {
	i.startDiscovery(ctx, time.Second)

	defer wg.Done()
	for {
		select {
		// Receive the synchrnization event with details regarding the mesh deployment
		case ev := <-i.evCh:
			log.Printf("Received Istio Event - %s\n", ev)
		case err := <-i.errCh:
			log.Printf("Received Error from Istio Sync - %s\n", err)
		case <-quit:
			return nil
		}
	}
}

// startDiscovery uses a poll based model to periodically (every syncPeriod) probe the istio controlplane.
// If the controlplane is deployed and responsive the info regarding the mesh is wrapped up and relayed
// over the "event" channel.
// However, if an error occurs during the probe, the error is reported back to the synchronizer
// over the "error" channel.
func (i *Istio) startDiscovery(ctx context.Context, syncPeriod time.Duration) {
	log.Println("Starting Istio synchronizer")
	tick := time.NewTicker(syncPeriod).C
	go func() {
		for {
			select {
			case <-tick:
				// FIXME (Nitish Malhotra) : This needs enhancement
				ver, err := i.cli.GetIstioVersions(ctx, i.namespace)
				if err != nil {
					i.errCh <- err
				} else {
					ev := &event{
						evType: models.Discovery,
						info:   ver,
					}
					// Process the discovery event and feed it to the
					// fingerprinting pipeline
					i.Process(ctx, ev)
				}
			}
		}

	}()
}

// Process the event based on its type
func (i *Istio) Process(ctx context.Context, event models.Event) {
	switch event.Type() {
	case models.Discovery:
		// Feed it to the processing pipeline
		i.processPipeline(ctx, event)
	}
}

// processPipeline passes the event through the processing pipeline
func (i *Istio) processPipeline(ctx context.Context, event models.Event) {
	i.evCh <- event
}
