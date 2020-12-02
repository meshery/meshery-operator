package broker

import (
	"github.com/layer5io/meshery-operator/pkg/broker/nats"
)

type Message struct {
	Type   string
	Object interface{}
}

type PublishInterface interface {
	Publish(string, interface{}) error
	PublishWithCallback(string, string, interface{}) error
}

type SubscribeInterface interface {
	Subscribe(string, string) error
	SubscribeWithHandler(string, string) error
}

type Handler interface {
	PublishInterface
	SubscribeInterface
}

const (
	NATSKey = "nats"
)

func New(kind string, url string) (Handler, error) {
	var broker Handler
	switch kind {
	case NATSKey:
		return nats.New(url)
	}
	return broker, nil
}
