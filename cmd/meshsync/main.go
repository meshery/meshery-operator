package main

import (
	"meshery-operator/internal/controller"
	"meshery-operator/pkg/kube"
	"meshery-operator/pkg/meshsync/istio"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	kubeconfig = kingpin.Flag("kubeconfig", "Path to kubernetes cluster's config file (usually at ~/.kube/config)").
		Envar("KUBECONFIG").String()
)

func main() {
	kingpin.Parse()

	kcli, err := kube.NewClient(kubeconfig)
	kingpin.FatalIfError(err, "failed to create Kubernetes client-go clientset")

	// Instantiate each of the relevant/supported mesh synchronizers here

	// Instantiate Istio synchronizer
	istioSync, err := istio.New(kcli)
	kingpin.FatalIfError(err, "failed to instantiate new Istio synchronizer")

	// Create a new meshsync controller that takes a list of synchronizers
	ctrl, err := controller.New(istioSync)
	kingpin.FatalIfError(err, "failed to instantiate meshsync controller")

	// OS interrupt signal channel
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// quit or stop channel
	quit := make(chan struct{})

	// Signal handler
	go func() {
		<-sigs
		quit <- struct{}{}
	}()

	// Start each of the synchronizers and block
	kingpin.FatalIfError(ctrl.Run(quit), "failed to start controller")
}
