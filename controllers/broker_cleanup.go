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
	"strings"
	"time"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	brokerpackage "github.com/meshery/meshery-operator/pkg/broker"
	meshsyncpackage "github.com/meshery/meshery-operator/pkg/meshsync"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	brokerMigrationRetryInterval = time.Minute
	brokerDeletionRetryInterval  = 5 * time.Second
)

// cleanupObsoleteBrokerResources prunes only the previous broker builder's
// StatefulSet and Service. Other child kinds, Secrets and PVCs are untouched.
// The result distinguishes slow migration polling from fast deletion confirmation;
// zero means no migration cleanup remains.
func (r *BrokerReconciler) cleanupObsoleteBrokerResources(ctx context.Context, broker *mesheryv1alpha1.Broker) (time.Duration, error) {
	if broker.DeletionTimestamp != nil || broker.UID == "" {
		return 0, nil
	}
	obsolete, err := r.legacyBrokerChildren(ctx, broker)
	if err != nil || len(obsolete) == 0 {
		return 0, err
	}
	pendingDeletion, allDeleting := false, true
	for _, object := range obsolete {
		if object.GetDeletionTimestamp() != nil {
			pendingDeletion = true
		} else {
			allDeleting = false
		}
	}
	// Confirmation does not require another migration check once every
	// candidate is already deleting. No new delete is requested on this path.
	if allDeleting {
		return brokerDeletionRetryInterval, nil
	}
	ready, err := r.brokerMigrationReady(ctx, broker)
	if err != nil {
		return 0, err
	}
	if !ready {
		if pendingDeletion {
			// Confirm the in-flight child promptly while retaining any child
			// whose migration gate is still closed.
			return brokerDeletionRetryInterval, nil
		}
		return brokerMigrationRetryInterval, nil
	}
	pending, err := r.deleteLegacyBrokerChildren(ctx, broker, obsolete)
	if pending {
		return brokerDeletionRetryInterval, err
	}
	return 0, err
}

// legacyBrokerChildren lists eligible kinds and verifies identity before pruning.
func (r *BrokerReconciler) legacyBrokerChildren(ctx context.Context, broker *mesheryv1alpha1.Broker) ([]client.Object, error) {
	options := []client.ListOption{client.InNamespace(broker.Namespace), client.MatchingLabels(brokerpackage.BrokerLabel)}
	statefulSets := &appsv1.StatefulSetList{}
	if err := r.List(ctx, statefulSets, options...); err != nil {
		return nil, err
	}
	services := &corev1.ServiceList{}
	if err := r.List(ctx, services, options...); err != nil {
		return nil, err
	}
	var obsolete []client.Object
	for i := range statefulSets.Items {
		if isLegacyBrokerChild(broker, &statefulSets.Items[i]) {
			obsolete = append(obsolete, &statefulSets.Items[i])
		}
	}
	for i := range services.Items {
		if isLegacyBrokerChild(broker, &services.Items[i]) {
			obsolete = append(obsolete, &services.Items[i])
		}
	}
	return obsolete, nil
}

// deleteLegacyBrokerChildren requests guarded deletion and retries until API objects disappear.
func (r *BrokerReconciler) deleteLegacyBrokerChildren(ctx context.Context, broker *mesheryv1alpha1.Broker, obsolete []client.Object) (bool, error) {
	for _, object := range obsolete {
		if object.GetDeletionTimestamp() != nil {
			continue
		}
		// UID prevents deleting a replacement; resourceVersion also protects an
		// ownership/label change between our cached read and the API deletion.
		uid, resourceVersion := object.GetUID(), object.GetResourceVersion()
		if resourceVersion == "" {
			return true, nil
		}
		err := r.Delete(ctx, object, client.Preconditions{UID: &uid, ResourceVersion: &resourceVersion})
		if apierrors.IsNotFound(err) {
			continue
		}
		if apierrors.IsConflict(err) {
			// Re-list and revalidate the candidate on the next reconciliation.
			return true, nil
		}
		if err != nil {
			return true, err
		}
		kind := brokerChildKind(object)
		r.Log.Info("Requested obsolete broker resource deletion", "kind", kind, "namespace", object.GetNamespace(), "name", object.GetName())
		if r.Recorder != nil {
			r.Recorder.Eventf(broker, corev1.EventTypeNormal, "ObsoleteBrokerResourceCleanup", "Requested deletion of obsolete %s %s/%s", kind, object.GetNamespace(), object.GetName())
		}
	}
	// Delete is asynchronous: retry until the old API objects are absent.
	return true, nil
}

// brokerChildKind limits migration candidates to the historical workload and Service kinds.
func brokerChildKind(object client.Object) string {
	switch object.(type) {
	case *appsv1.StatefulSet:
		return "StatefulSet"
	case *corev1.Service:
		return "Service"
	default:
		return ""
	}
}

// isLegacyBrokerChild requires historical identity and ownership while protecting desired resources.
func isLegacyBrokerChild(broker *mesheryv1alpha1.Broker, object client.Object) bool {
	// The old builders used Broker.Name, including custom names. A different
	// name is not a verified legacy resource even if somebody set these labels.
	if broker.UID == "" || object.GetUID() == "" || object.GetNamespace() != broker.Namespace || object.GetName() != broker.Name || brokerChildKind(object) == "" {
		return false
	}
	for key, value := range brokerpackage.BrokerLabel {
		if object.GetLabels()[key] != value {
			return false
		}
	}
	if !isControlledByBroker(broker, object) {
		return false
	}
	// Explicitly protect current identities even when a Broker is named
	// meshery-nats and a current object still carries historical labels.
	for _, desired := range brokerpackage.GetObjects(broker) {
		current, ok := desired.(client.Object)
		if ok && current.GetName() == object.GetName() && brokerChildKind(current) == brokerChildKind(object) {
			return false
		}
	}
	return true
}

// isControlledByBroker verifies controller identity independently of the served owner version.
func isControlledByBroker(broker *mesheryv1alpha1.Broker, object client.Object) bool {
	owner := metav1.GetControllerOf(object)
	if owner == nil || owner.UID != broker.UID || owner.Name != broker.Name || owner.Kind != "Broker" {
		return false
	}
	version, err := schema.ParseGroupVersion(owner.APIVersion)
	if err != nil || version.Group != mesheryv1alpha1.GroupVersion.Group {
		return false
	}
	return true
}

// brokerMigrationReady checks workload rollout state, not live NATS connections.
// External/custom clients cannot be inferred from Kubernetes owner references.
func (r *BrokerReconciler) brokerMigrationReady(ctx context.Context, broker *mesheryv1alpha1.Broker) (bool, error) {
	if broker.Status.Endpoint.Internal == "" {
		return false, nil
	}
	ready, err := r.replacementBrokerRolledOut(ctx, broker)
	if err != nil || !ready {
		return false, err
	}
	meshsyncs := &mesheryv1alpha1.MeshSyncList{}
	if err := r.List(ctx, meshsyncs); err != nil {
		return false, err
	}
	for i := range meshsyncs.Items {
		m := &meshsyncs.Items[i]
		if m.Spec.Broker.Native == (mesheryv1alpha1.NativeMeshsyncBroker{}) {
			continue
		}
		targetNamespace := m.Spec.Broker.Native.Namespace
		if targetNamespace == "" {
			targetNamespace = m.Namespace
		}
		if (m.Spec.Broker.Native.Name != "" && m.Spec.Broker.Native.Name != broker.Name) || targetNamespace != broker.Namespace {
			continue
		}
		ready, err := r.meshSyncBrokerRolledOut(ctx, broker, m)
		if err != nil || !ready {
			return false, err
		}
	}
	return true, nil
}

// replacementBrokerRolledOut reads desired StatefulSets and waits for their current rollout.
func (r *BrokerReconciler) replacementBrokerRolledOut(ctx context.Context, broker *mesheryv1alpha1.Broker) (bool, error) {
	for _, desired := range brokerpackage.GetObjects(broker) {
		want, ok := desired.(*appsv1.StatefulSet)
		if !ok {
			continue
		}
		current := &appsv1.StatefulSet{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(want), current); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if !replacementBrokerRolloutComplete(broker, current, *want.Spec.Replicas) {
			return false, nil
		}
	}
	return true, nil
}

// replacementBrokerRolloutComplete requires the owned workload to reach the desired revision and replicas.
func replacementBrokerRolloutComplete(broker *mesheryv1alpha1.Broker, current *appsv1.StatefulSet, replicas int32) bool {
	return isControlledByBroker(broker, current) && current.DeletionTimestamp == nil &&
		current.Status.ObservedGeneration >= current.Generation && current.Status.CurrentRevision != "" &&
		current.Status.CurrentRevision == current.Status.UpdateRevision &&
		current.Status.Replicas == replicas && current.Status.UpdatedReplicas == replicas && current.Status.ReadyReplicas == replicas
}

// meshSyncBrokerRolledOut verifies that an owned consumer has rolled out the replacement configuration.
func (r *BrokerReconciler) meshSyncBrokerRolledOut(ctx context.Context, broker *mesheryv1alpha1.Broker, m *mesheryv1alpha1.MeshSync) (bool, error) {
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: m.Namespace, Name: m.Name}, deployment); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	owner := metav1.GetControllerOf(deployment)
	if owner == nil || owner.UID != m.UID || deployment.DeletionTimestamp != nil {
		return false, nil
	}
	copy := m.DeepCopy()
	copy.Status.PublishingTo = natsURL(broker.Status.Endpoint.Internal)
	want := meshsyncpackage.GetServerObject(copy, brokerpackage.AuthSecretName).(*appsv1.Deployment)
	if !meshSyncDeploymentRolloutComplete(deployment, *want.Spec.Replicas) {
		return false, nil
	}

	for _, container := range want.Spec.Template.Spec.Containers {
		if !brokerEnvMatches(container, deployment.Spec.Template.Spec.Containers) {
			return false, nil
		}
	}
	return true, nil
}

// meshSyncDeploymentRolloutComplete rejects stale, unavailable or reported terminating replicas.
func meshSyncDeploymentRolloutComplete(deployment *appsv1.Deployment, replicas int32) bool {
	return deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == replicas &&
		deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.UpdatedReplicas == replicas && deployment.Status.Replicas == replicas &&
		deployment.Status.ReadyReplicas == replicas && deployment.Status.AvailableReplicas == replicas &&
		(deployment.Status.TerminatingReplicas == nil || *deployment.Status.TerminatingReplicas == 0)
}

// brokerEnvMatches checks unique broker variables and token-expansion order,
// ignoring unrelated environment entries. Duplicate names are ambiguous because
// later definitions can override a value that matched earlier in the list.
func brokerEnvMatches(want corev1.Container, containers []corev1.Container) bool {
	for _, current := range containers {
		if current.Name != want.Name {
			continue
		}
		tokenIndex, urlIndex := -1, -1
		urlNeedsToken := false
		for _, expected := range want.Env {
			if expected.Name != "BROKER_URL" && expected.Name != "NATS_TOKEN" {
				continue
			}
			matchIndex := -1
			for i, actual := range current.Env {
				if actual.Name != expected.Name {
					continue
				}
				if matchIndex >= 0 || !brokerEnvVarMatches(actual, expected) {
					return false
				}
				matchIndex = i
			}
			if matchIndex < 0 {
				return false
			}
			if expected.Name == "NATS_TOKEN" {
				tokenIndex = matchIndex
			} else {
				urlIndex = matchIndex
				urlNeedsToken = strings.Contains(expected.Value, "$(NATS_TOKEN)")
			}
		}
		// Kubernetes expands $(VAR) only from earlier environment entries.
		return !urlNeedsToken || (tokenIndex >= 0 && tokenIndex < urlIndex)
	}
	return false
}

// brokerEnvVarMatches compares literal URLs and Secret references without treating
// an omitted optional flag differently from false. A different source or an
// optional token Secret still fails the migration gate.
func brokerEnvVarMatches(actual, expected corev1.EnvVar) bool {
	if actual.Value != expected.Value {
		return false
	}
	if expected.ValueFrom == nil {
		return actual.ValueFrom == nil
	}
	if actual.ValueFrom == nil || expected.ValueFrom.SecretKeyRef == nil {
		return false
	}
	source := actual.ValueFrom
	if source.FieldRef != nil || source.ResourceFieldRef != nil || source.ConfigMapKeyRef != nil || source.FileKeyRef != nil {
		return false
	}
	current, want := source.SecretKeyRef, expected.ValueFrom.SecretKeyRef
	return current != nil && current.Name == want.Name && current.Key == want.Key &&
		(current.Optional != nil && *current.Optional) == (want.Optional != nil && *want.Optional)
}
