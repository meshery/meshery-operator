package broker

import (
	"github.com/layer5io/meshery-operator/pkg/broker/nats"
)

type PublishInterface interface {
	Publish(string, interface{})
	PublishWithCallback(string, string, interface{})
}

type SubscribeInterface interface {
	Subscribe(string, string)
	SubscribeWithHandler(string, string)
}

type Broker interface {
	PublishInterface
	SubscribeInterface
}

const (
	NATSKey = "nats"
)

func New(kind string, url string) (Broker, error) {
	var broker Broker
	switch kind {
	case NATSKey:
		return nats.New(url)
	}
	return broker, nil
}
