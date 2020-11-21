package informers

import (
	"log"

	v1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func (i *Istio) DestinationRuleInformer() cache.SharedIndexInformer {
	// get informer
	DestinationRuleInformer := i.client.GetDestinationRuleInformer().Informer()

	// register event handlers
	DestinationRuleInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addDestinationRule,
			UpdateFunc: updateDestinationRule,
			DeleteFunc: deleteDestinationRule,
		},
	)

	return DestinationRuleInformer
}

func deleteDestinationRule(obj interface{}) {
	DestinationRule := obj.(*v1beta1.DestinationRule)
	log.Printf("DestinationRule Named: %s - deleted", DestinationRule.Name)
}

func addDestinationRule(obj interface{}) {
	DestinationRule := obj.(*v1beta1.DestinationRule)
	log.Printf("DestinationRule Named: %s - added", DestinationRule.Name)
}

func updateDestinationRule(new interface{}, old interface{}) {
	DestinationRule := new.(*v1beta1.DestinationRule)
	log.Printf("DestinationRule Named: %s - updated", DestinationRule.Name)
}
