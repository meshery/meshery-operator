/*
Copyright 2020 Layer5, Inc.

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	// pkgmeshsync "github.com/layer5io/meshery-operator/pkg/meshsync"
	appsv1 "k8s.io/api/apps/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
)

// MeshSyncReconciler reconciles a MeshSync object
type MeshSyncReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=meshery.layer5.io,resources=meshsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meshery.layer5.io,resources=meshsyncs/status,verbs=get;update;patch
func (r *MeshSyncReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {

	ctx := context.Background()
	log := r.Log.WithValues("meshsync", req.NamespacedName)
	log.Info("Reconcillation")

	meshsync := &mesheryv1alpha1.MeshSync{}
	if err := r.Get(ctx, req.NamespacedName, meshsync); err != nil {
		log.Error(err, "unable to fetch MeshSync")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: meshsync.Name, Namespace: meshsync.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		_ = createDeployment(meshsync, r.Scheme)
		// log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		// err = r.Create(ctx, dep)
		// if err != nil {
		// 	log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		// 	return ctrl.Result{}, err
		// }
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *MeshSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mesheryv1alpha1.MeshSync{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
