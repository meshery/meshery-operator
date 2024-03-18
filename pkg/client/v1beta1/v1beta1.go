package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

var (
	ParameterCodec runtime.ParameterCodec
)

type CoreInterface interface {
	RESTClient() rest.Interface
	BrokersGetter
	MeshSyncsGetter
}

type CoreClient struct {
	restClient rest.Interface
}

func New(c rest.Interface, codec runtime.ParameterCodec) *CoreClient {
	ParameterCodec = codec
	return &CoreClient{
		restClient: c,
	}
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
