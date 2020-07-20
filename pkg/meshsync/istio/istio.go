package istio

import (
	"context"
	"fmt"
	"meshery-operator/pkg/kube"
	"meshery-operator/pkg/meshsync/models"
	"sync"
	"time"

	"github.com/prometheus/common/log"
	ikube "istio.io/istio/pkg/kube"
	"istio.io/pkg/version"
	// apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type Istio struct {
	cli       ikube.ExtendedClient
	namespace string

	kcli *kube.Client
	// Other fields TBD
	isDeployed bool
}

func New(kcli *kube.Client, ns string) (*Istio, error) {
	extendedClient, err := ikube.NewExtendedClient(ikube.BuildClientCmd(kcli.Kubeconfig(), ""), "")
	if err != nil {
		return nil, err
	}
	return &Istio{
		cli:       extendedClient,
		namespace: ns,
		kcli:      kcli,
	}, nil
}

func (i *Istio) Synchronize(ctx context.Context, wg *sync.WaitGroup, quit <-chan struct{}) error {
	// TODO: (Adheip) fingerprintEventsCh := fingerprinter.Start() -> For istio we start the sharedinformer
	log.Info("Starting Istio synchronizer")
	evCh := i.startSync(ctx)

	defer wg.Done()
	for {
		select {
		case ev := <-evCh:
			if ev.Version() != nil && ev.Error() == nil {
				i.isDeployed = true
				fmt.Printf("[Event] Istio Controlplane := version %s\n", *ev.Version())
			} else {
				// Log error
			}

		// Pull fingerprint Events from go-channel
		// case ev := <-fingerprintEventsCh:
		// lookup for istio.io string match in CRD ev.objects
		// Make decision and set isDeployed flag as required
		// if found i.isDeployed = true
		case <-quit:
			return nil
		}
	}
}

func (i *Istio) IsDeployed() bool {
	return i.isDeployed
}

func (i *Istio) startSync(ctx context.Context) <-chan models.Info {
	out := make(chan models.Info, 1)
	ch := time.NewTicker(time.Second).C

	go func() {
		for {
			select {
			case <-ch:
				ver, err := i.cli.GetIstioVersions(ctx, i.namespace)
				out <- &event{
					info: ver,
					err:  err,
				}
			}
		}

	}()
	return out
}

type event struct {
	info *version.MeshInfo
	err  error
}

func (ev *event) Version() *string {
	var v string
	for _, comp := range *ev.info {
		v = v + fmt.Sprintf("[component] %s [version] %s, ", comp.Component, comp.Info.LongForm())
	}
	return &v
}

func (ev *event) Details() models.MeshInfo {
	info := models.MeshInfo{
		Version: ev.Version(),
	}

	return info
}

func (ev *event) Error() error {
	return ev.err
}
