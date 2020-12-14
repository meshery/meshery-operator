package service

import (
	"context"
	"log"

	"github.com/golang/protobuf/ptypes/empty"
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

func (s *Service) Initialize(config *rest.Config) error {
	// Configure discovery
	dclient, err := discovery.NewClient(config)
	if err != nil {
		s.Logger.Error(ErrNewDiscovery(err))
		log.Printf("Error in creating discovery client: %s", err)
		return ErrNewDiscovery(err)
	}

	// Configure informers
	iclient, err := informers.NewClient(config)
	if err != nil {
		s.Logger.Error(ErrNewInformer(err))
		log.Printf("Error in creating informer client: %s", err)
		return err
	}

	err = cluster.Setup(dclient, s.Broker, iclient)
	if err != nil {
		s.Logger.Error(ErrSetupCluster(err))
		return err
	}

	err = istio.Setup(dclient, s.Broker, iclient)
	if err != nil {
		s.Logger.Error(ErrSetupIstio(err))
		return err
	}

	return nil
}
