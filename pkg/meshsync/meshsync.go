package meshsync

import (
	"log"
	"os"
	"time"

	"github.com/layer5io/meshery-operator/pkg/broker"
	"github.com/layer5io/meshery-operator/pkg/meshsync/service"
	utils "github.com/layer5io/meshkit/utils/kubernetes"
)

func main() {
	kubeconfig, err := utils.DetectKubeConfig()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	br, err := broker.New(broker.NATSKey, "<server-url>")
	if err != nil {
		log.Printf("Error while creating broker: %s", err)
		os.Exit(1)
	}

	err = service.Discover(kubeconfig, br)
	if err != nil {
		log.Printf("Error while discovery: %s", err)
		os.Exit(1)
	}

	err = service.Start(&service.Service{
		Name:      "meshsync",
		Port:      "11000",
		Version:   "v0.0.1-alpha3",
		StartedAt: time.Now(),
	})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
