package nats

import nats "github.com/nats-io/nats.go"

// it need three things
// 1 subject
// 2 message to send
// 3 server URL

// Nats will implement Nats subscribe and publish functionality
type Nats struct {
	ec *nats.EncodedConn
}

// NewNats - constructor
func NewNats(serverURL string) (*Nats, error) {
	nc, err := nats.Connect(serverURL)
	if err != nil {
		return nil, err
	}
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}
	return &Nats{ec: ec}, nil
}

// Publish - to publish messages
func (n *Nats) Publish(subject string, message interface{}) error {
	err := n.ec.Publish(subject, message)
	return err
}

// Subscribe - for subscribing messages
// arguments:
// subject - a string to which it should subsribe
// callback - a function that will be called everytime a message is recieved
// return:
// sub - if we want to unsubscribe in future
// err - error if any
func (n *Nats) Subscribe(subject string, callback func(interface{})) (*nats.Subscription, error) {
	sub, err := n.ec.Subscribe(subject, callback)
	return sub, err
}
