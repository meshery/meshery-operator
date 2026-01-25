/*
Copyright Meshery Authors

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
	"time"

	"github.com/go-logr/logr"
	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	brokerpackage "github.com/meshery/meshery-operator/pkg/broker"
	meshsyncpackage "github.com/meshery/meshery-operator/pkg/meshsync"
	"github.com/meshery/meshery-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	util "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MeshSyncReconciler reconciles a MeshSync object
type MeshSyncReconciler struct {
	client.Client
	KubeConfig *rest.Config
	Clientset  *kubernetes.Clientset
	Scheme     *runtime.Scheme
	Log        logr.Logger
}

const (
	meshsyncFinalizer = "meshsync.meshery.io/finalizer"
)

// +kubebuilder:rbac:groups=meshery.io,resources=meshsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meshery.io,resources=meshsyncs/status,verbs=get;update;patch

// Reconcile reconciles the MeshSync resource
func (r *MeshSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log
	log = log.WithValues("controller", "MeshSync")
	log = log.WithValues("namespace", req.NamespacedName)
	log.Info("Reconciling MeshSync")
	baseResource := &mesheryv1alpha1.MeshSync{}

	// Check if resource exists
	err := r.Get(ctx, req.NamespacedName, baseResource)
	if err != nil {
		return r.handleGetError(log, err)
	}
	// resource deletion
	if baseResource.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, log, baseResource)
	}

	if result, err := r.ensureFinalizer(ctx, log, baseResource); err != nil || result.RequeueAfter > 0 {
		return result, err
	}
	return r.performReconciliation(ctx, log, baseResource, req)

}

func (r *MeshSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mesheryv1alpha1.MeshSync{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// performReconciliation performs the main meshsync reconciliation logic
func (r *MeshSyncReconciler) performReconciliation(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.MeshSync, req ctrl.Request) (ctrl.Result, error) {
	// Set initial status to processing
	if err := r.updateStatusCondition(ctx, baseResource, "Processing", v1.ConditionTrue, "Reconciling", "Reconciling meshsync"); err != nil {
		log.Error(err, "Failed to update meshsync status to Processing")
	}

	// Get broker configuration
	if err := r.reconcileBrokerConfig(ctx, baseResource); err != nil {
		log.Error(err, "Failed to reconcile broker config")
		return ctrl.Result{}, ErrReconcileMeshsync(err)
	}

	// Deploy meshsync resources
	if result, err := r.deployMeshsyncResources(ctx, log, baseResource, req); err != nil {
		return result, err
	}

	// Check meshsync health
	if err := r.checkMeshsyncHealth(ctx, baseResource); err != nil {
		log.Info("Health check failed, will retry in 5 seconds", "error", err)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// Update status to Ready
	if err := r.updateStatusCondition(ctx, baseResource, "Ready", v1.ConditionTrue, "ReconciliationSuccessful", "meshsync reconciled successfully"); err != nil {
		log.Error(err, "Failed to update meshsync status to Ready")
	}

	// Patch the meshsync resource status
	return r.patchMeshsyncStatus(ctx, log, baseResource)
}

func (r *MeshSyncReconciler) deployMeshsyncResources(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.MeshSync, req ctrl.Request) (ctrl.Result, error) {
	result, err := r.reconcileMeshsync(ctx, true, baseResource, req)
	if err != nil {
		err = ErrReconcileMeshsync(err)
		log.Error(err, "Meshsync reconciliation failed")
		_ = r.updateStatusCondition(ctx, baseResource, "Failed", v1.ConditionFalse, "ReconciliationFailed", "Meshsync reconciliation failed")
		return ctrl.Result{}, err
	}
	return result, nil
}

func (r *MeshSyncReconciler) checkMeshsyncHealth(ctx context.Context, baseResource *mesheryv1alpha1.MeshSync) error {
	if err := meshsyncpackage.CheckHealth(ctx, baseResource, r.Client); err != nil {
		err = ErrCheckHealth(err)
		_ = r.updateStatusCondition(ctx, baseResource, "Failed", v1.ConditionFalse, "HealthCheckFailed", "Meshsync health check failed")
		return err
	}
	return nil
}

func (r *MeshSyncReconciler) patchMeshsyncStatus(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.MeshSync) (ctrl.Result, error) {
	patch, err := utils.Marshal(baseResource)
	if err != nil {
		err = ErrUpdateResource(err)
		log.Error(err, "unable to update meshsync resource")
		return ctrl.Result{}, err
	}
	err = r.Status().Patch(ctx, baseResource, client.RawPatch(types.MergePatchType, []byte(patch)))
	if err != nil {
		if kubeerror.IsConflict(err) {
			log.Error(err, "conflict while patching meshsync resource, requeuing to retry")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		err = ErrUpdateResource(err)
		log.Error(err, "unable to patch meshsync resource")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *MeshSyncReconciler) ensureFinalizer(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.MeshSync) (ctrl.Result, error) {
	if util.ContainsFinalizer(baseResource, meshsyncFinalizer) {
		return ctrl.Result{}, nil
	}

	log.Info("Adding finalizer to meshsync")
	util.AddFinalizer(baseResource, meshsyncFinalizer)
	if err := r.Update(ctx, baseResource); err != nil {
		if kubeerror.IsConflict(err) {
			log.Error(err, "Conflict while adding finalizer, requeuing to retry")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}
	log.Info("Finalizer added successfully, requeuing for next reconcile")
	return ctrl.Result{RequeueAfter: 0}, nil

}

func (r *MeshSyncReconciler) handleDeletion(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.MeshSync) (ctrl.Result, error) {
	if !util.ContainsFinalizer(baseResource, meshsyncFinalizer) {
		log.Info("meshsync resource is being deleted (no finalizers)")
		return ctrl.Result{}, nil
	}
	log.Info("Executing meshsync finalizers")
	if err := r.Cleanup(ctx, baseResource); err != nil {
		// update the status condition
		log.Error(err, "Failed to cleanup meshsync")
		_ = r.updateStatusCondition(ctx, baseResource, "Cleanup", v1.ConditionFalse, "CleanupFailed", err.Error())
		return ctrl.Result{}, ErrDeleteMeshsync(err)
	}

	return r.removeFinalizer(ctx, log, baseResource)
}

// removeFinalizer removes the finalizer from the meshsync resource
func (r *MeshSyncReconciler) removeFinalizer(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.MeshSync) (ctrl.Result, error) {
	util.RemoveFinalizer(baseResource, meshsyncFinalizer)
	if err := r.Update(ctx, baseResource); err != nil {
		if kubeerror.IsConflict(err) {
			log.Error(err, "Conflict while removing finalizer, requeuing to retry")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}
	log.Info("Broker finalizers executed successfully, resource will be deleted")
	return ctrl.Result{}, nil
}

func (r *MeshSyncReconciler) handleGetError(log logr.Logger, err error) (ctrl.Result, error) {
	if kubeerror.IsNotFound(err) {
		log.Info("meshsync resource not found, likely already deleted")
		return ctrl.Result{}, nil
	}
	log.Error(err, "unable to get meshsync resource")
	return ctrl.Result{}, ErrGetMeshsync(err)
}

func (r *MeshSyncReconciler) Cleanup(ctx context.Context, baseResource *mesheryv1alpha1.MeshSync) error {
	log := r.Log.WithValues("meshsync", baseResource.Name, "namespace", baseResource.Namespace)
	objects := meshsyncpackage.GetObjects(baseResource)
	log.Info("Cleaning up meshsync resources")
	for _, object := range objects {
		log.Info("Deleting meshsync object", "kind", object.GetObjectKind().GroupVersionKind().Kind, "name", object.GetName())
		err := r.Delete(ctx, object)
		if err != nil {
			if kubeerror.IsConflict(err) {
				log.V(1).Error(err, "Object not found, skipping", "name", object.GetName())
				// skip this is normal
				continue
			}
			log.Error(err, "Unable to delete meshsync object", "name", object.GetName())
			return ErrDeleteMeshsync(err)
		}
	}
	log.Info("Successfully cleaned up meshsync resources")
	return nil
}

// updateStatusCondition updates the status condition and handles conflicts gracefully
func (r *MeshSyncReconciler) updateStatusCondition(ctx context.Context, meshsync *mesheryv1alpha1.MeshSync, conditionType string, status v1.ConditionStatus, reason, message string) error {
	log := r.Log.WithValues("meshsync", meshsync.Name, "namespace", meshsync.Namespace)

	meta.SetStatusCondition(&meshsync.Status.Conditions, v1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: meshsync.GetGeneration(),
	})
	if err := r.Status().Update(ctx, meshsync); err != nil {
		if kubeerror.IsConflict(err) {
			log.V(1).Error(err, "Conflict while updating status, will retry on next reconcile")
			return nil
		}
		return err
	}
	return nil
}

func (r *MeshSyncReconciler) reconcileBrokerConfig(ctx context.Context, baseResource *mesheryv1alpha1.MeshSync) error {
	brokerresource := &mesheryv1alpha1.Broker{}
	nullNativeResource := mesheryv1alpha1.NativeMeshsyncBroker{}
	if baseResource.Spec.Broker.Native != nullNativeResource {
		brokerresource.Namespace = baseResource.Spec.Broker.Native.Namespace
		brokerresource.Name = baseResource.Spec.Broker.Native.Name
		err := brokerpackage.GetEndpoint(ctx, brokerresource, r.Client, r.KubeConfig.Host)
		if err != nil {
			return ErrGetEndpoint(err)
		}
		baseResource.Status.PublishingTo = brokerresource.Status.Endpoint.Internal
	} else if baseResource.Spec.Broker.Custom.URL != "" {
		// Add handler for custom broker config
		baseResource.Status.PublishingTo = baseResource.Spec.Broker.Custom.URL
	}

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
	switch {
	case err != nil && kubeerror.IsNotFound(err) && enable:
		_ = util.SetControllerReference(baseResource, object, r.Scheme)
		er := r.Create(ctx, object)
		if er != nil {
			return ctrl.Result{}, ErrCreateMeshsync(er)
		}
		r.Log.Info("Meshsync created successfully")
		// .Owns will trigger a reconcile for the object
		return ctrl.Result{}, nil
	case err != nil && enable:
		return ctrl.Result{}, ErrGetMeshsync(err)
	case err == nil && !kubeerror.IsNotFound(err) && !enable:
		er := r.Delete(ctx, object)
		if er != nil {
			return ctrl.Result{}, ErrDeleteMeshsync(er)
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{}, nil
}
