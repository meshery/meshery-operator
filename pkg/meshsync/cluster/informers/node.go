package informers

import (
	"context"
	"log"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func GetNodeInformer(clientset *kubernetes.Clientset) cache.SharedInformer {
	sharedInformer := cache.NewSharedInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return clientset.CoreV1().Nodes().List(
					context.Background(),
					metav1.ListOptions{},
				)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return clientset.CoreV1().Nodes().Watch(
					context.Background(),
					metav1.ListOptions{},
				)
			},
		},
		&v1.Node{},
		time.Second,
	)

	// adding event handler
	sharedInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: nodeDeleted,
		},
	)

	// to get store run
	// sharedInformer.GetStore()
	// sharedInformer.Run(wait.NeverStop)
	return sharedInformer
}

func nodeDeleted(obj interface{}) {
	pod := obj.(*v1.Node)
	log.Println(pod.Name)
}
