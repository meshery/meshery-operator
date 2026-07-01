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
	"github.com/meshery/meshery-operator/pkg/metrics"
	"github.com/meshery/meshery-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	util "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	// meshsyncControllerName labels this controller's reconciliation metrics.
	meshsyncControllerName = "meshsync"
	// meshsyncFieldManager is the stable Server-Side Apply field manager for the
	// MeshSync Deployment.
	meshsyncFieldManager = "meshery-operator-meshsync"
)

// +kubebuilder:rbac:groups=meshery.io,resources=meshsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meshery.io,resources=meshsyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=meshery.io,resources=meshsyncs/finalizers,verbs=update
// +kubebuilder:rbac:groups=meshery.io,resources=brokers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile reconciles the MeshSync resource
func (r *MeshSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reconcileErr error) {
	start := time.Now()
	defer func() {
		metrics.ReconcileTotal.WithLabelValues(meshsyncControllerName).Inc()
		metrics.ReconcileDuration.WithLabelValues(meshsyncControllerName).Observe(time.Since(start).Seconds())
		if reconcileErr != nil {
			metrics.ReconcileErrors.WithLabelValues(meshsyncControllerName).Inc()
		}
	}()

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

	if res, err := r.ensureFinalizer(ctx, log, baseResource); err != nil || res.RequeueAfter > 0 {
		return res, err
	}
	return r.performReconciliation(ctx, log, baseResource)

}

func (r *MeshSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Watch Broker objects so a change to a referenced Broker's endpoint
	// re-enqueues the MeshSync that consumes it, propagating the new BROKER_URL
	// without manual intervention (WS-3 §4.3 #15, WS-4 §6.2.4).
	return ctrl.NewControllerManagedBy(mgr).
		For(&mesheryv1alpha1.MeshSync{}).
		Owns(&appsv1.Deployment{}).
		Watches(&mesheryv1alpha1.Broker{}, handler.EnqueueRequestsFromMapFunc(r.meshsyncsForBroker)).
		Complete(r)
}

// meshsyncsForBroker maps a Broker event to reconcile requests for every
// MeshSync that references it as a native broker.
func (r *MeshSyncReconciler) meshsyncsForBroker(ctx context.Context, obj client.Object) []reconcile.Request {
	broker, ok := obj.(*mesheryv1alpha1.Broker)
	if !ok {
		return nil
	}

	var meshsyncs mesheryv1alpha1.MeshSyncList
	if err := r.List(ctx, &meshsyncs); err != nil {
		r.Log.Error(err, "unable to list MeshSync resources for Broker watch", "broker", broker.Name)
		return nil
	}

	var requests []reconcile.Request
	for i := range meshsyncs.Items {
		native := meshsyncs.Items[i].Spec.Broker.Native
		if native.Name == broker.Name && native.Namespace == broker.Namespace {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      meshsyncs.Items[i].Name,
				Namespace: meshsyncs.Items[i].Namespace,
			}})
		}
	}
	return requests
}

// performReconciliation performs the main meshsync reconciliation logic
func (r *MeshSyncReconciler) performReconciliation(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.MeshSync) (ctrl.Result, error) {
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
	if result, err := r.deployMeshsyncResources(ctx, log, baseResource); err != nil {
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

func (r *MeshSyncReconciler) deployMeshsyncResources(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.MeshSync) (ctrl.Result, error) {
	if err := r.reconcileMeshsync(ctx, baseResource); err != nil {
		log.Error(err, "Meshsync reconciliation failed")
		_ = r.updateStatusCondition(ctx, baseResource, "Failed", v1.ConditionFalse, "ReconciliationFailed", "Meshsync reconciliation failed")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
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
		err = ErrMarshal(err)
		log.Error(err, "unable to marshal meshsync resource")
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
	log.Info("MeshSync finalizers executed successfully, resource will be deleted")
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
			if kubeerror.IsNotFound(err) {
				log.V(1).Info("Object not found, skipping", "name", object.GetName())
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
		// Inject the NATS token (if the broker uses token auth) into the URL so
		// MeshSync can authenticate: nats://<token>@host:port.
		token := r.brokerToken(ctx, brokerresource.Namespace)
		baseResource.Status.PublishingTo = natsURL(token, brokerresource.Status.Endpoint.Internal)
	} else if baseResource.Spec.Broker.Custom.URL != "" {
		// Add handler for custom broker config
		baseResource.Status.PublishingTo = baseResource.Spec.Broker.Custom.URL
	}

	return nil
}

// brokerToken returns the NATS auth token from the broker's meshery-nats-auth
// Secret, or "" when the broker runs without token auth.
func (r *MeshSyncReconciler) brokerToken(ctx context.Context, namespace string) string {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: brokerpackage.AuthSecretName, Namespace: namespace}, secret); err != nil {
		return ""
	}
	return string(secret.Data["token"])
}

// natsURL builds a scheme-qualified broker URL, embedding the token as userinfo
// when present.
func natsURL(token, hostPort string) string {
	if hostPort == "" {
		return ""
	}
	if token == "" {
		return "nats://" + hostPort
	}
	return "nats://" + token + "@" + hostPort
}

// reconcileMeshsync drives the MeshSync Deployment to its desired state with a
// single Server-Side Apply, mirroring the Broker controller. SSA avoids the
// read-modify-DeepEqual hot-loop and lets the API server keep its defaulted
// fields (WS-3 §4.3 #13, §6.2.2).
func (r *MeshSyncReconciler) reconcileMeshsync(ctx context.Context, baseResource *mesheryv1alpha1.MeshSync) error {
	desired := meshsyncpackage.GetServerObject(baseResource)
	desired.SetNamespace(baseResource.Namespace)
	if err := util.SetControllerReference(baseResource, desired, r.Scheme); err != nil {
		return ErrReconcileMeshsync(err)
	}
	if err := r.apply(ctx, desired); err != nil {
		return ErrReconcileMeshsync(err)
	}
	return nil
}

// apply performs a Server-Side Apply of the MeshSync Deployment using a stable
// field manager.
func (r *MeshSyncReconciler) apply(ctx context.Context, obj client.Object) error {
	gvk, err := apiutil.GVKForObject(obj, r.Scheme)
	if err != nil {
		return err
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	return r.Patch(ctx, obj, client.Apply, client.FieldOwner(meshsyncFieldManager), client.ForceOwnership)
}
