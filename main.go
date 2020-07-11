package main

import (
	"meshery-controller/controller"
	"meshery-controller/utils"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	AllNamespaces = ""
)

func main() {
	//clientset := utils.GetKubeClientset()
	istioclientset := utils.GetIstioClientset()
	// create the virtualService2 watcher
	deployListWatcher := cache.NewListWatchFromClient(istioclientset.NetworkingV1alpha3().RESTClient(), "virtualservices", AllNamespaces, fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	// initialize the controller object
	controller := controller.NewController(istioclientset, queue, deployListWatcher)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	select {}
}
