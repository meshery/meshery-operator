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
	"encoding/json"
	"slices"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Finalizers that operator releases up to and including 1.0.6 added to every
// Broker and MeshSync. They are no longer added: every object either controller
// creates carries a controller ownerReference to its CR (the NATS ConfigMap,
// client + headless Services, StatefulSet and auth Secret for a Broker; the
// Deployment and any mirrored auth Secret for a MeshSync), all namespaced in
// the CR's own namespace, and nothing cluster-scoped is created. Kubernetes
// garbage collection therefore removes everything the finalizers used to
// delete, and keeps doing so when the operator itself is gone.
//
// The finalizers were not just redundant but harmful: when the operator is
// removed in the same pass as its CRs (a Helm uninstall of the operator chart,
// which Meshery Server performs on disconnect or when a connection switches to
// embedded MeshSync), nothing is left to run them. The CRs then stay
// Terminating forever, their owned NATS and MeshSync workloads are never
// collected, and a later reinstall adopts the Terminating CRs only for the new
// operator to finalize them away.
//
// The constants remain so the reconcilers can migrate existing CRs by
// stripping them (see stripLegacyFinalizer).
const (
	legacyBrokerFinalizer   = "broker.meshery.io/finalizer"
	legacyMeshsyncFinalizer = "meshsync.meshery.io/finalizer"

	// legacyFinalizerConflictRetry matches the retry the controllers use for
	// their other conflicting metadata and status writes.
	legacyFinalizerConflictRetry = 5 * time.Second
)

// migrateLegacyFinalizer is the reconcile step that strips a legacy finalizer
// (see stripLegacyFinalizer). stop reports that the reconcile must end now,
// returning result and err:
//   - the CR is already gone (NotFound): nothing is left to reconcile, and
//     carrying on with the stale copy would re-apply children owned by a
//     deleted UID;
//   - a conflict: the cached copy is stale, so retry shortly rather than
//     report a reconcile error;
//   - any other failure: a structured error naming the CR, so
//     controller-runtime backs off and retries. Callers also record it on
//     the CR's Ready condition.
//
// Otherwise (finalizer absent, or removed) the reconcile continues with obj,
// which reflects the server's copy after a successful patch.
func migrateLegacyFinalizer(ctx context.Context, c client.Client, log logr.Logger, obj client.Object, finalizer string) (stop bool, result ctrl.Result, err error) {
	removed, err := stripLegacyFinalizer(ctx, c, obj, finalizer)
	switch {
	case apierrors.IsNotFound(err):
		log.Info("Resource already deleted while removing legacy finalizer", "finalizer", finalizer)
		return true, ctrl.Result{}, nil
	case apierrors.IsConflict(err):
		log.V(1).Info("Conflict while removing legacy finalizer, requeuing to retry", "finalizer", finalizer)
		return true, ctrl.Result{RequeueAfter: legacyFinalizerConflictRetry}, nil
	case err != nil:
		err = ErrRemoveLegacyFinalizer(finalizer, obj.GetNamespace(), obj.GetName(), err)
		log.Error(err, "Failed to remove legacy finalizer", "finalizer", finalizer)
		return true, ctrl.Result{}, err
	}
	if removed {
		log.Info("Removed legacy finalizer; owned objects are cleaned up by garbage collection", "finalizer", finalizer)
	}
	return false, ctrl.Result{}, nil
}

// stripLegacyFinalizer removes a finalizer an earlier operator release added,
// on live and Terminating CRs alike. For a Terminating CR it is the step that
// lets deletion complete; garbage collection then removes the owned objects
// through their ownerReferences, so no explicit cleanup runs first.
//
// It reports whether it patched the object. The JSON merge patch carries the
// object's resourceVersion as an optimistic lock, so a finalizer list another
// writer changed concurrently is never clobbered: the API server rejects the
// stale patch with a Conflict, which the caller retries. API errors, including
// NotFound for an object deleted meanwhile, are returned unwrapped for the
// caller to classify. On success obj reflects the server's response; on
// failure it is left untouched.
func stripLegacyFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) (bool, error) {
	if !controllerutil.ContainsFinalizer(obj, finalizer) {
		return false, nil
	}
	// Clone before filtering: DeleteFunc works in place and obj must stay
	// unchanged if the patch fails. A cloned non-empty slice stays non-nil
	// after filtering, so an emptied list marshals as [] rather than null.
	remaining := slices.DeleteFunc(slices.Clone(obj.GetFinalizers()), func(f string) bool { return f == finalizer })
	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"finalizers":      remaining,
			"resourceVersion": obj.GetResourceVersion(),
		},
	})
	if err != nil {
		return false, err
	}
	if err := c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch)); err != nil {
		return false, err
	}
	return true, nil
}
