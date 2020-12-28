/*


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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	brokerpackage "github.com/layer5io/meshery-operator/pkg/broker"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
)

// BrokerReconciler reconciles a Broker object
type BrokerReconciler struct {
	client.Client
	Clientset *kubernetes.Clientset
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=meshery.layer5.io,resources=brokers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meshery.layer5.io,resources=brokers/status,verbs=get;update;patch

func (r *BrokerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("namespace", req.NamespacedName)
	log.Info("Reconcillation")
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
		return ctrl.Result{}, ErrReconcileBroker(err)
	}

	// Check if Broker controller started
	err = brokerpackage.CheckHealth(ctx, baseResource, r.Clientset)
	if err != nil {
		return ctrl.Result{Requeue: true}, ErrCheckHealth(err)
	}

	// Get broker endpoint
	err = brokerpackage.GetEndpoint(ctx, baseResource, r.Clientset)
	if err != nil {
		return ctrl.Result{}, ErrGetEndpoint(err)
	}

	return result, nil
}

func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mesheryv1alpha1.Broker{}).
		Complete(r)
}

func (r *BrokerReconciler) reconcileBroker(ctx context.Context, enable bool, baseResource *mesheryv1alpha1.Broker, req ctrl.Request) (ctrl.Result, error) {
	object := brokerpackage.GetObjects(baseResource)[brokerpackage.ServerObject]
	err := r.Get(ctx,
		types.NamespacedName{
			Name:      baseResource.Name,
			Namespace: baseResource.Namespace,
		},
		object,
	)
	if err != nil && kubeerror.IsNotFound(err) && enable {
		er := r.Create(ctx, object)
		if er != nil {
			return ctrl.Result{}, ErrCreateMeshsync(er)
		}
		_ = ctrl.SetControllerReference(baseResource, object, r.Scheme)
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

	return ctrl.Result{}, nil
}
