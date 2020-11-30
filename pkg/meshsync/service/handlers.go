package service

import (
	"context"
	"log"

	"github.com/golang/protobuf/ptypes/empty"
	broker "github.com/layer5io/meshery-operator/pkg/broker"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
	informers "github.com/layer5io/meshery-operator/pkg/informers"
	"github.com/layer5io/meshery-operator/pkg/meshsync/cluster"
	"github.com/layer5io/meshery-operator/pkg/meshsync/meshes/istio"
	proto "github.com/layer5io/meshery-operator/pkg/meshsync/proto"
	controller "github.com/layer5io/meshkit/protobuf/controller"

	"k8s.io/client-go/rest"
)

func (s *Service) Info(context.Context, *empty.Empty) (*controller.ControllerInfo, error) {
	return &controller.ControllerInfo{
		Name:    s.Name,
		Version: s.Version,
	}, nil
}

func (s *Service) Health(context.Context, *empty.Empty) (*controller.ControllerHealth, error) {
	return &controller.ControllerHealth{
		Status: controller.ControllerStatus_RUNNING,
	}, nil
}

func (s *Service) Sync(context.Context, *proto.Request) (*proto.Response, error) {
	return &proto.Response{
		Result: &proto.Response_Message{
			Message: "ok",
		},
	}, nil
}

func Discover(config *rest.Config, broker broker.Broker) error {

	// Configure discovery
	client, err := discovery.NewClient(config)
	if err != nil {
		log.Printf("Couldnot create client: %s", err)
		return err
	}

	err = cluster.StartDiscovery(client, broker)
	if err != nil {
		return err
	}

	err = istio.StartDiscovery(client, broker)
	if err != nil {
		return err
	}

	return nil
}

// StartInformer - run informer
func StartInformers(config *rest.Config, broker broker.Broker) error {

	// Configure discovery
	client, err := informers.NewClient(config)
	if err != nil {
		log.Printf("Couldnot create informer client: %s", err)
		return err
	}

	log.Println("start cluster informers")
	cluster.StartInformer(client, broker)

	log.Println("start istio informers")
	istio.StartInformer(client, broker)

	return nil
}
