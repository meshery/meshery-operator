package meshsync

import (
	"fmt"
	"os"
	"time"

	"github.com/layer5io/meshery-operator/pkg/broker"
	"github.com/layer5io/meshery-operator/pkg/meshsync/service"
	"github.com/layer5io/meshkit/logger"
	"github.com/layer5io/meshkit/utils/kubernetes"
)

var (
	serviceName = "meshsync"
)

func Main() {
	// Initialize Logger instance
	log, err := logger.New(serviceName, logger.Options{
		Format: logger.SyslogLogFormat,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Initialize Kubeconfig
	kubeconfig, err := kubernetes.DetectKubeConfig()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Initialize Broker instance
	br, err := broker.New(broker.NATSKey, "<server-url>")
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Initialize service by running pre-defined tasks
	sHandler := &service.Service{
		Name:      "meshsync",
		Port:      "11000",
		Version:   "v0.0.1-alpha3",
		StartedAt: time.Now(),
		Logger:    log,
		Broker:    br,
	}

	err = sHandler.Initialize(kubeconfig)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Start GRPC server
	log.Info("Adaptor Listening at port: ", sHandler.Port)
	err = service.Start(sHandler)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
