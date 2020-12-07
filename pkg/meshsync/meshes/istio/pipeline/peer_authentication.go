package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// PeerAuthentication will implement step interface for PeerAuthentications
type PeerAuthentication struct {
	pipeline.StepContext
	// clients
	client *discovery.Client
	broker broker.Handler
}

// NewPeerAuthentication - constructor
func NewPeerAuthentication(client *discovery.Client, broker broker.Handler) *PeerAuthentication {
	return &PeerAuthentication{
		client: client,
		broker: broker,
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

		// processing
		for _, peerAuthentication := range peerAuthentications {
			// publishing discovered peerAuthentication
			err := pa.broker.Publish(Subject, broker.Message{
				Type:   "PeerAuthentication",
				Object: peerAuthentication,
			})
			if err != nil {
				log.Printf("Error publishing peer authentication named %s", peerAuthentication.Name)
			} else {
				log.Printf("Published peer authentication named %s", peerAuthentication.Name)
			}
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
