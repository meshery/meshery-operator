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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	util "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	brokerpackage "github.com/layer5io/meshery-operator/pkg/broker"
	meshsyncpackage "github.com/layer5io/meshery-operator/pkg/meshsync"
	"github.com/layer5io/meshkit/utils"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
)

// MeshSyncReconciler reconciles a MeshSync object
type MeshSyncReconciler struct {
	client.Client
	KubeConfig *rest.Config
	Clientset  *kubernetes.Clientset
	Log        logr.Logger
	Scheme     *runtime.Scheme
}

// +kubebuilder:rbac:groups=meshery.layer5.io,resources=meshsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meshery.layer5.io,resources=meshsyncs/status,verbs=get;update;patch
func (r *MeshSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log
	log = log.WithValues("controller", "MeshSync")
	log = log.WithValues("namespace", req.NamespacedName)
	log.Info("Reconciling MeshSync")
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

	// Get broker configuration
	err = r.reconcileBrokerConfig(ctx, baseResource)
	if err != nil {
		return ctrl.Result{}, ErrReconcileMeshsync(err)
	}

	// Check if Meshsync controller running
	result, err := r.reconcileMeshsync(ctx, true, baseResource, req)
	if err != nil {
		return ctrl.Result{}, ErrReconcileMeshsync(err)
	}

	// Patch the meshsync resource
	patch, err := utils.Marshal(baseResource)
	if err != nil {
		return ctrl.Result{}, ErrUpdateResource(err)
	}

	err = r.Status().Patch(ctx, baseResource, client.RawPatch(types.MergePatchType, []byte(patch)))
	if err != nil {
		return ctrl.Result{}, ErrUpdateResource(err)
	}

	return result, nil
}

func (r *MeshSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mesheryv1alpha1.MeshSync{}).
		Complete(r)
}

func (r *MeshSyncReconciler) Cleanup() error {
	objects := meshsyncpackage.GetObjects(&mesheryv1alpha1.MeshSync{
		ObjectMeta: v1.ObjectMeta{
			Name:      "meshery-meshsync",
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

func (r *MeshSyncReconciler) reconcileBrokerConfig(ctx context.Context, baseResource *mesheryv1alpha1.MeshSync) error {
	brokerresource := &mesheryv1alpha1.Broker{}
	nullNativeResource := mesheryv1alpha1.NativeMeshsyncBroker{}
	if baseResource.Spec.Broker.Native != nullNativeResource {
		brokerresource.ObjectMeta.Namespace = baseResource.Spec.Broker.Native.Namespace
		brokerresource.ObjectMeta.Name = baseResource.Spec.Broker.Native.Name
		err := brokerpackage.GetEndpoint(ctx, r.Log, brokerresource, r.Clientset, r.KubeConfig.Host)
		if err != nil {
			return ErrGetEndpoint(err)
		}
		baseResource.Status.PublishingTo = brokerresource.Status.Endpoint.Internal
	}

	// Add handler for custom broker config

	return nil
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
		_ = util.SetControllerReference(baseResource, object, r.Scheme)
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

	return ctrl.Result{}, nil
}
