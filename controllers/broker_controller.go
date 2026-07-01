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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// BrokerReconciler reconciles a Broker object
type BrokerReconciler struct {
	client.Client
	KubeConfig *rest.Config
	Clientset  *kubernetes.Clientset
	Scheme     *runtime.Scheme
	Log        logr.Logger
}

const (
	brokerFinalizer = "broker.meshery.io/finalizer"
	// brokerControllerName labels this controller's reconciliation metrics.
	brokerControllerName = "broker"
	// brokerFieldManager is the stable Server-Side Apply field manager for all
	// broker-owned objects. It must not change across releases, or the API
	// server would treat previously-applied fields as orphaned.
	brokerFieldManager = "meshery-operator-broker"
)

// +kubebuilder:rbac:groups=meshery.io,resources=brokers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=meshery.io,resources=brokers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=meshery.io,resources=brokers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;configmaps;secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is the main reconciliation loop
func (r *BrokerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reconcileErr error) {
	start := time.Now()
	defer func() {
		metrics.ReconcileTotal.WithLabelValues(brokerControllerName).Inc()
		metrics.ReconcileDuration.WithLabelValues(brokerControllerName).Observe(time.Since(start).Seconds())
		if reconcileErr != nil {
			metrics.ReconcileErrors.WithLabelValues(brokerControllerName).Inc()
		}
	}()

	log := r.Log.WithValues("controller", "Broker", "namespace", req.NamespacedName)
	log.Info("Reconciling broker")

	baseResource := &mesheryv1alpha1.Broker{}
	err := r.Get(ctx, req.NamespacedName, baseResource)
	if err != nil {
		return r.handleGetError(log, err)
	}

	// resource deletion
	if baseResource.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, log, baseResource)
	}

	// finalizer exists
	if res, err := r.ensureFinalizer(ctx, log, baseResource); err != nil || res.RequeueAfter > 0 {
		return res, err
	}

	// main reconciliation
	return r.performReconciliation(ctx, log, baseResource)
}

// handleGetError handles errors from fetching the broker resource
func (r *BrokerReconciler) handleGetError(log logr.Logger, err error) (ctrl.Result, error) {
	if kubeerror.IsNotFound(err) {
		log.Info("Broker resource not found, likely already deleted")
		return ctrl.Result{}, nil
	}
	log.Error(err, "Failed to get broker resource")
	return ctrl.Result{}, ErrGetBroker(err)
}

// handleDeletion handles the deletion of a broker resource with finalizers
func (r *BrokerReconciler) handleDeletion(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.Broker) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(baseResource, brokerFinalizer) {
		log.Info("Broker is being deleted (no finalizers)")
		return ctrl.Result{}, nil
	}

	log.Info("Executing broker finalizers")
	if err := r.Cleanup(ctx, baseResource); err != nil {
		log.Error(err, "Failed to cleanup broker")
		_ = r.updateStatusCondition(ctx, baseResource, "Cleanup", v1.ConditionFalse, "CleanupFailed", err.Error())
		return ctrl.Result{}, ErrDeleteBroker(err)
	}

	return r.removeFinalizer(ctx, log, baseResource)
}

// removeFinalizer removes the finalizer from the broker resource
func (r *BrokerReconciler) removeFinalizer(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.Broker) (ctrl.Result, error) {
	if removed := controllerutil.RemoveFinalizer(baseResource, brokerFinalizer); !removed {
		log.Info("Finalizer not found, likely already removed")
		return ctrl.Result{}, nil
	}
	if err := r.Update(ctx, baseResource); err != nil {
		if kubeerror.IsConflict(err) {
			log.Error(err, "Conflict while removing finalizer, requeuing to retry")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}
	log.Info("Broker finalizers executed successfully, resource will be deleted")
	return ctrl.Result{RequeueAfter: 0}, nil
}

// ensureFinalizer adds the finalizer if it doesn't exist
func (r *BrokerReconciler) ensureFinalizer(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.Broker) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(baseResource, brokerFinalizer) {
		return ctrl.Result{}, nil
	}

	log.Info("Adding finalizer to broker")
	controllerutil.AddFinalizer(baseResource, brokerFinalizer)
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

// performReconciliation performs the main broker reconciliation logic
func (r *BrokerReconciler) performReconciliation(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.Broker) (ctrl.Result, error) {
	// Set initial status to processing
	if err := r.updateStatusCondition(ctx, baseResource, "Processing", v1.ConditionTrue, "Reconciling", "Reconciling broker"); err != nil {
		log.Error(err, "Failed to update broker status to Processing")
	}

	// Deploy broker resources
	if result, err := r.deployBrokerResources(ctx, log, baseResource); err != nil {
		return result, err
	}

	// Check broker health
	if err := r.checkBrokerHealth(ctx, baseResource); err != nil {
		log.Info("Health check failed, will retry in 5 seconds", "error", err)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	// Get broker endpoint
	if err := r.getBrokerEndpoint(ctx, log, baseResource); err != nil {
		return ctrl.Result{}, err
	}

	// Update status to Ready
	if err := r.updateStatusCondition(ctx, baseResource, "Ready", v1.ConditionTrue, "ReconciliationSuccessful", "Broker reconciled successfully"); err != nil {
		log.Error(err, "Failed to update broker status to Ready")
	}

	// Patch the broker resource status
	return r.patchBrokerStatus(ctx, log, baseResource)
}

// deployBrokerResources deploys the broker controller and resources
func (r *BrokerReconciler) deployBrokerResources(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.Broker) (ctrl.Result, error) {
	if err := r.reconcileBroker(ctx, baseResource); err != nil {
		log.Error(err, "Broker reconciliation failed")
		_ = r.updateStatusCondition(ctx, baseResource, "Ready", v1.ConditionFalse, "ReconciliationFailed", err.Error())
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// checkBrokerHealth checks if the broker controller is healthy
func (r *BrokerReconciler) checkBrokerHealth(ctx context.Context, baseResource *mesheryv1alpha1.Broker) error {
	if err := brokerpackage.CheckHealth(ctx, baseResource, r.Client); err != nil {
		_ = r.updateStatusCondition(ctx, baseResource, "Ready", v1.ConditionFalse, "HealthCheckFailed", err.Error())
		return ErrCheckHealth(err)
	}
	return nil
}

// getBrokerEndpoint retrieves the broker endpoint
func (r *BrokerReconciler) getBrokerEndpoint(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.Broker) error {
	if err := brokerpackage.GetEndpoint(ctx, baseResource, r.Client, r.KubeConfig.Host); err != nil {
		err = ErrGetEndpoint(err)
		log.Error(err, "Unable to get the broker endpoint")
		_ = r.updateStatusCondition(ctx, baseResource, "Ready", v1.ConditionFalse, "EndpointFailed", err.Error())
		return err
	}
	return nil
}

// patchBrokerStatus patches the broker resource status
func (r *BrokerReconciler) patchBrokerStatus(ctx context.Context, log logr.Logger, baseResource *mesheryv1alpha1.Broker) (ctrl.Result, error) {
	patch, err := utils.Marshal(baseResource)
	if err != nil {
		err = ErrMarshal(err)
		log.Error(err, "Unable to marshal broker resource")
		return ctrl.Result{}, err
	}

	err = r.Status().Patch(ctx, baseResource, client.RawPatch(types.MergePatchType, []byte(patch)))
	if err != nil {
		if kubeerror.IsConflict(err) {
			log.Error(err, "Conflict while patching broker resource, requeuing to retry")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		err = ErrUpdateResource(err)
		log.Error(err, "Unable to patch broker resource")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// updateStatusCondition updates the status condition and handles conflicts gracefully
func (r *BrokerReconciler) updateStatusCondition(ctx context.Context, broker *mesheryv1alpha1.Broker, conditionType string, status v1.ConditionStatus, reason, message string) error {
	log := r.Log.WithValues("broker", broker.Name, "namespace", broker.Namespace)

	meta.SetStatusCondition(&broker.Status.Conditions, v1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: broker.GetGeneration(),
	})

	if err := r.Status().Update(ctx, broker); err != nil {
		if kubeerror.IsConflict(err) {
			log.V(1).Error(err, "Conflict while updating status, will retry on next reconcile")
			return nil
		}
		return err
	}
	return nil
}

func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Own the Service and ConfigMaps in addition to the StatefulSet so that a
	// Service type/port edit or a late LoadBalancer IP assignment re-enqueues the
	// Broker and the endpoint is recomputed (WS-3 §4.3 #15, WS-4 §6.2.1).
	return ctrl.NewControllerManagedBy(mgr).
		For(&mesheryv1alpha1.Broker{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (r *BrokerReconciler) Cleanup(ctx context.Context, baseResource *mesheryv1alpha1.Broker) error {
	log := r.Log.WithValues("broker", baseResource.Name, "namespace", baseResource.Namespace)
	log.Info("Cleaning up broker resources")
	objects := brokerpackage.GetObjects(baseResource)
	for _, object := range objects {
		log.Info("Deleting broker object", "kind", object.GetObjectKind().GroupVersionKind().Kind, "name", object.GetName())
		err := r.Delete(ctx, object)
		if err != nil {
			// check if this request is due to object not found
			if kubeerror.IsNotFound(err) {
				log.V(1).Info("Object not found, skipping", "name", object.GetName())
				continue
			}
			log.Error(err, "Unable to delete broker object", "name", object.GetName())
			return ErrDeleteBroker(err)
		}
	}
	log.Info("Successfully cleaned up broker resources")
	return nil
}

// reconcileBroker drives every broker-owned object to its desired state with a
// single Server-Side Apply per object. SSA reconciles only the operator-owned
// fields and leaves server-defaulted fields (clusterIP, nodePort,
// healthCheckNodePort, StatefulSet/Pod defaults) untouched, which eliminates the
// read-modify-DeepEqual hot-loop the old sync helpers caused (WS-3 §4.3 #13,
// §6.2.2). Applying the full ordered object set every reconcile also removes the
// "return after first Create" partial-progress behaviour (#16).
func (r *BrokerReconciler) reconcileBroker(ctx context.Context, baseResource *mesheryv1alpha1.Broker) error {
	// The NATS StatefulSet reads its token from the meshery-nats-auth Secret, so
	// that Secret must exist before the pod starts. Generate it once and keep the
	// existing token on subsequent reconciles (SSA-applying a fresh random token
	// every reconcile would rotate it and drop client connections).
	if err := r.ensureAuthSecret(ctx, baseResource); err != nil {
		return err
	}
	for _, object := range brokerpackage.GetObjects(baseResource) {
		object.SetNamespace(baseResource.Namespace)
		if err := ctrl.SetControllerReference(baseResource, object, r.Scheme); err != nil {
			return ErrReconcileBroker(err)
		}
		if err := r.apply(ctx, object); err != nil {
			return ErrReconcileBroker(err)
		}
	}
	return nil
}

// ensureAuthSecret creates the meshery-nats-auth Secret with a freshly generated
// token the first time, and is a no-op once it exists so the token is stable.
func (r *BrokerReconciler) ensureAuthSecret(ctx context.Context, baseResource *mesheryv1alpha1.Broker) error {
	key := types.NamespacedName{Name: brokerpackage.AuthSecretName, Namespace: baseResource.Namespace}
	existing := &corev1.Secret{}
	err := r.Get(ctx, key, existing)
	if err == nil {
		return nil
	}
	if !kubeerror.IsNotFound(err) {
		return ErrGetBroker(err)
	}

	token, err := brokerpackage.GenerateToken()
	if err != nil {
		return ErrCreateBroker(err)
	}
	secret := brokerpackage.BuildAuthSecret(baseResource.Namespace, token)
	if err := ctrl.SetControllerReference(baseResource, secret, r.Scheme); err != nil {
		return ErrReconcileBroker(err)
	}
	if err := r.Create(ctx, secret); err != nil {
		if kubeerror.IsAlreadyExists(err) {
			return nil
		}
		return ErrCreateBroker(err)
	}
	return nil
}

// apply performs a Server-Side Apply of a single owned object using a stable
// field manager. The object's GroupVersionKind must be populated for SSA, so it
// is resolved from the scheme.
func (r *BrokerReconciler) apply(ctx context.Context, obj client.Object) error {
	gvk, err := apiutil.GVKForObject(obj, r.Scheme)
	if err != nil {
		return err
	}
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	return r.Patch(ctx, obj, client.Apply, client.FieldOwner(brokerFieldManager), client.ForceOwnership)
}
