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
	"k8s.io/client-go/kubernetes"
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
	Clientset *kubernetes.Clientset
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=meshery.layer5.io,resources=meshsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meshery.layer5.io,resources=meshsyncs/status,verbs=get;update;patch
func (r *MeshSyncReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("meshsync", req.NamespacedName)
	log.Info("Reconcillation")
	baseResource := &mesheryv1alpha1.MeshSync{}

	// Check if resource exists
	err := r.Get(ctx, req.NamespacedName, baseResource)
	if err != nil {
		if kubeerror.IsNotFound(err) {
			baseResource.Name = req.Name
			baseResource.Namespace = req.Namespace
			return r.reconcileMeshsync(ctx, false, baseResource, req)
		}
		return ctrl.Result{}, err
	}

	// Check if Meshsync controller running
	result, err := r.reconcileMeshsync(ctx, true, baseResource, req)
	if err != nil {
		return ctrl.Result{}, ErrReconcileMeshsync(err)
	}

	return result, nil
}

func (r *MeshSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mesheryv1alpha1.MeshSync{}).
		Complete(r)
}

func (r *MeshSyncReconciler) reconcileMeshsync(ctx context.Context, enable bool, baseResource *mesheryv1alpha1.MeshSync, req ctrl.Request) (ctrl.Result, error) {

	object := meshsyncpackage.GetObjects(baseResource)[meshsyncpackage.ServerObject]
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
