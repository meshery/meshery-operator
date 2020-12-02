package pipeline

import (
	"log"

	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// AuthorizationPolicy will implement step interface for AuthorizationPolicies
type AuthorizationPolicy struct {
	pipeline.StepContext
	client *discovery.Client
	broker broker.Handler
}

// NewAuthorizationPolicy - constructor
func NewAuthorizationPolicy(client *discovery.Client, broker broker.Handler) *AuthorizationPolicy {
	return &AuthorizationPolicy{
		client: client,
		broker: broker,
	}
}

// Exec - step interface
func (ap *AuthorizationPolicy) Exec(request *pipeline.Request) *pipeline.Result {
	// it will contain a pipeline to run
	log.Println("AuthorizationPolicy Discovery Started")

	for _, namespace := range Namespaces {
		authorizationPolicies, err := ap.client.ListAuthorizationPolicies(namespace)
		if err != nil {
			return &pipeline.Result{
				Error: err,
			}
		}

		// processing
		for _, authorizationPolicy := range authorizationPolicies {
			// publishing discovered authorizationPolicy
			err := ap.broker.Publish(Subject, broker.Message{
				Type:   "authorizationPolicy",
				Object: authorizationPolicy,
			})
			if err != nil {
				log.Printf("Error publishing authorizationPolicy named %s", authorizationPolicy.Name)
			} else {
				log.Printf("Published authorizationPolicy named %s", authorizationPolicy.Name)
			}
		}
	}

	// no data is feeded to future steps or stages
	return &pipeline.Result{
		Error: nil,
	}
}

// Cancel - step interface
func (ap *AuthorizationPolicy) Cancel() error {
	ap.Status("cancel step")
	return nil
}
