package nats

import nats "github.com/nats-io/nats.go"

// Nats will implement Nats subscribe and publish functionality
type Nats struct {
	ec *nats.EncodedConn
}

// New - constructor
func New(serverURL string) (*Nats, error) {
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

// PublishWithCallback - will implement the request-reply mechanisms
// Arguments:
// request - the subject to which publish a request
// reply - this string will be used by the replier to publish replies
// message - message send by the requestor to replier
// TODO Ques: After this the requestor have to subscribe to the reply subject
func (n *Nats) PublishWithCallback(request string, reply string, message interface{}) error {
	err := n.ec.PublishRequest(request, reply, message)
	return err
}

// Subscribe - for subscribing messages
// TODO Ques: Do we want to unsubscribe
// TODO will the method-user just subsribe, how will it handle the received messages?
func (n *Nats) Subscribe(subject string, queue string) error {
	// no handler
	// TODO there should be a callback that handler received messges
	_, err := n.ec.QueueSubscribe(subject, queue, func() {})
	return err
}

// SubscribeWithHandler - for handling request-reply protocol
// request is the subject to which the this thing is listening
// when there will be a request
func (n *Nats) SubscribeWithHandler(subject string, queue string) error {
	// no handler
	_, err := n.ec.QueueSubscribe(subject, queue, func() {})
	return err
}
