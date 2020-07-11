package controller

import (
	"fmt"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"
)

func (c *Controller) syncHandler(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("Error fetching obj %s from store failed due to %v\n", key, err)
		return err
	}
	if !exists {
		fmt.Printf("vs [%s]  does not exist \n", key)
	} else {
		fmt.Printf("virtual service [%s] \n", obj.(*v1alpha3.VirtualService).Name)
	}
	return nil

}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {

		c.queue.Forget(key)
		return
	}

	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("Error syncing pod %v: %v", key, err)
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	runtime.HandleError(err)
	klog.Infof("Dropping Deployemnt %q out of the queue: %v", key, err)
}
