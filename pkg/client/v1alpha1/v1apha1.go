package v1alpha1

import (
	"k8s.io/client-go/rest"
)

type CoreInterface interface {
	RESTClient() rest.Interface
	BrokersGetter
	MeshSyncsGetter
}

type CoreClient struct {
	restClient rest.Interface
}

func New(c rest.Interface) *CoreClient {
	return &CoreClient{c}
}

func (c *CoreClient) Brokers(namespace string) BrokerInterface {
	return newBrokers(c, namespace)
}

func (c *CoreClient) MeshSyncs(namespace string) MeshSyncInterface {
	return newMeshSyncs(c, namespace)
}

func (c *CoreClient) RESTClient() rest.Interface {
	return c.restClient
}
