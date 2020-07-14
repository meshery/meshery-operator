package istio

import (
	"meshery-operator/pkg/kube"
	"sync"

	"github.com/prometheus/common/log"
	// apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type Istio struct {
	kcli *kube.Client
	// Other fields TBD
	isDeployed bool
}

func New(kcli *kube.Client) (*Istio, error) {
	// TODO: (Adheip) Create new SharedIndexInformer for CRDs (can we use a filter as well for istio.io string???)
	// Maybe abstract this out into a fingerprint component/controller/service in pkg/meshsync/fingerprint/istio.go
	// FingerPrinter could be an interface with an Identify() method, implemented by each service mesh synchronizer

	return &Istio{
		kcli: kcli,
	}, nil
}

func (i *Istio) Synchronize(wg *sync.WaitGroup, quit <-chan struct{}) error {
	// TODO: (Adheip) fingerprintEventsCh := fingerprinter.Start() -> For istio we start the sharedinformer

	log.Info("Starting Istio synchronizer")
	defer wg.Done()
	for {
		select {
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
