package pipeline

import (
	"log"

	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	"github.com/myntra/pipeline"
)

// AuthorizationPolicy will implement step interface for AuthorizationPolicies
type AuthorizationPolicy struct {
	pipeline.StepContext
	client *discovery.Client
}

// NewAuthorizationPolicy - constructor
func NewAuthorizationPolicy(client *discovery.Client) *AuthorizationPolicy {
	return &AuthorizationPolicy{
		client: client,
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
			log.Printf("Discovered authorization policy named %s in namespace %s", authorizationPolicy.Name, namespace)
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
