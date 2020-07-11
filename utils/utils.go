package utils

import (
	"log"
	"os"

	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func client() *rest.Config {
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
	if err != nil {
		log.Printf("error in getting Kubeconfig: %v", err)
	}
	return conf
}

// GetKubeClientset shall return a kubernetes clientset
func GetKubeClientset() *kubernetes.Clientset {
	conf := client()

	cs, err := kubernetes.NewForConfig(conf)
	if err != nil {
		log.Printf("error in getting clientset from Kubeconfig: %v", err)
	}

	return cs
}

// GetIstioClientset shall return the istio client
func GetIstioClientset() *versionedclient.Clientset {
	conf := client()

	ic, err := versionedclient.NewForConfig(conf)
	if err != nil {
		log.Fatalf("Failed to create istio client: %s", err)
	}
	return ic
}

// TypeConv shall return a string
func TypeConv(i interface{}) string {
	a := i.(string)
	return a
}
