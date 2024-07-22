/*
Copyright 2023 Layer5, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	brokerpackage "github.com/layer5io/meshery-operator/pkg/broker"
	"github.com/layer5io/meshery-operator/pkg/utils"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
)

// BrokerReconciler reconciles a Broker object
type BrokerReconciler struct {
	client.Client
	KubeConfig *rest.Config
	Clientset  *kubernetes.Clientset
	Log        logr.Logger
	Scheme     *runtime.Scheme
}

// +kubebuilder:rbac:groups=meshery.layer5.io,resources=brokers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meshery.layer5.io,resources=brokers/status,verbs=get;update;patch

func (r *BrokerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log
	log = log.WithValues("controller", "Broker")
	log = log.WithValues("namespace", req.NamespacedName)
	log.Info("Reconciling broker")
	baseResource := &mesheryv1alpha1.Broker{}

	// Check if resource exists
	err := r.Get(ctx, req.NamespacedName, baseResource)
	if err != nil {
		if kubeerror.IsNotFound(err) {
			baseResource.Name = req.Name
			baseResource.Namespace = req.Namespace
			return r.reconcileBroker(ctx, false, baseResource, req)
		}
		return ctrl.Result{}, err
	}

	// Check if Broker controller deployed
	result, err := r.reconcileBroker(ctx, true, baseResource, req)
	if err != nil {
		err = ErrReconcileBroker(err)
		r.Log.Error(err, "broker reconcilation failed")
		return ctrl.Result{}, err
	}

	// Check if Broker controller started
	err = brokerpackage.CheckHealth(ctx, baseResource, r.Clientset)
	if err != nil {
		return ctrl.Result{Requeue: true}, ErrCheckHealth(err)
	}

	// Get broker endpoint
	err = brokerpackage.GetEndpoint(ctx, baseResource, r.Clientset, r.KubeConfig.Host)
	if err != nil {
		err = ErrGetEndpoint(err)
		r.Log.Error(err, "unable to get the broker endpoint")
		return ctrl.Result{}, err
	}

	// Patch the broker resource
	patch, err := utils.Marshal(baseResource)
	if err != nil {
		err = ErrUpdateResource(err)
		r.Log.Error(err, "unable to update broker resource")
		return ctrl.Result{}, err
	}

	err = r.Status().Patch(ctx, baseResource, client.RawPatch(types.MergePatchType, []byte(patch)))
	if err != nil {
		err = ErrUpdateResource(err)
		r.Log.Error(err, "unable to update broker resource")
		return ctrl.Result{}, err
	}

	return result, nil
}

func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mesheryv1alpha1.Broker{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}

func (r *BrokerReconciler) Cleanup() error {
	objects := brokerpackage.GetObjects(&mesheryv1alpha1.Broker{
		ObjectMeta: v1.ObjectMeta{
			Name:      "meshery-broker",
			Namespace: "meshery",
		},
	})
	for _, object := range objects {
		err := r.Delete(context.TODO(), object)
		if err != nil {
			return ErrDeleteMeshsync(err)
		}
	}
	return nil
}

func (r *BrokerReconciler) reconcileBroker(ctx context.Context, enable bool, baseResource *mesheryv1alpha1.Broker, req ctrl.Request) (ctrl.Result, error) {
	objects := brokerpackage.GetObjects(baseResource)
	for _, object := range objects {
		object.SetNamespace(baseResource.Namespace)
		err := r.Get(ctx,
			types.NamespacedName{
				Name:      object.GetName(),
				Namespace: object.GetNamespace(),
			},
			object,
		)
		if err != nil && kubeerror.IsNotFound(err) && enable {
			_ = ctrl.SetControllerReference(baseResource, object, r.Scheme)
			er := r.Create(ctx, object)
			if er != nil {
				return ctrl.Result{}, ErrCreateMeshsync(er)
			}
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil && enable {
			return ctrl.Result{}, ErrGetMeshsync(err)
		} else if err == nil && !kubeerror.IsNotFound(err) && !enable {
			er := r.Delete(ctx, object)
			if er != nil {
				return ctrl.Result{}, ErrDeleteMeshsync(er)
			}
			return ctrl.Result{Requeue: true}, nil
		}
	}

	return ctrl.Result{}, nil
}
