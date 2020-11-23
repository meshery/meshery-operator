package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// PeerAuthentication will implement step interface for PeerAuthentications
type PeerAuthentication struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
}

// NewPeerAuthentication - constructor
func NewPeerAuthentication(client *discovery.Client) *PeerAuthentication {
	return &PeerAuthentication{
		client: client,
	}
}

// Exec - step interface
func (pa *PeerAuthentication) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("PeerAuthentication Discovery Started")

	for _, namespace := range Namespaces {
		peerAuthentications, err := pa.client.ListPeerAuthentications(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// process PeerAuthentications
		for _, peerAuthentication := range peerAuthentications {
			log.Printf("Discovered PeerAuthentication named %s in namespace %s", peerAuthentication.Name, namespace)
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (pa *PeerAuthentication) Cancel() error {
	pa.Status("cancel step")
	return nil
}
