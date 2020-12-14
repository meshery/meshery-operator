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
	meshsyncpackage "github.com/layer5io/meshery-operator/pkg/meshsync"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
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

	// Check if resource exists
	baseResource := &mesheryv1alpha1.MeshSync{}
	err := r.Get(ctx, req.NamespacedName, baseResource)
	if err != nil {
		if kubeerror.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Meshsync resource not found")
		return ctrl.Result{}, err
	}

	// Check if controllers running
	// Meshsync
	meshsync := meshsyncpackage.GetResource(baseResource)
	err = r.Get(ctx, types.NamespacedName{Name: baseResource.Name, Namespace: baseResource.Namespace}, meshsync)
	if err != nil && kubeerror.IsNotFound(err) {
		dep := meshsyncpackage.CreateResource(baseResource, r.Scheme)
		log.Error(err, "Failed to get Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, ErrGetMeshsync(err)
	}

	return ctrl.Result{}, nil
}

func (r *MeshSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mesheryv1alpha1.MeshSync{}).
		Complete(r)
}
