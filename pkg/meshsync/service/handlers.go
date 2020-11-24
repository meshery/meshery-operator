package service

import (
	"context"
	"log"

	"github.com/golang/protobuf/ptypes/empty"
	discovery "github.com/layer5io/meshery-operator/pkg/discovery"
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

func Discover(config *rest.Config) error {

	// Configure discovery
	client, err := discovery.NewClient(config)
	if err != nil {
		log.Printf("Couldnot create client: %s", err)
		return err
	}

	err = cluster.StartDiscovery(client)
	if err != nil {
		return err
	}

	err = istio.StartDiscovery(client)
	if err != nil {
		return err
	}

	return nil
}
