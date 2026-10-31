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

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	brokerpackage "github.com/meshery/meshery-operator/pkg/broker"
	meshsyncpackage "github.com/meshery/meshery-operator/pkg/meshsync"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// cleanupObsoleteBrokerResources prunes only the previous broker builder's
// StatefulSet and Service. Other child kinds, Secrets and PVCs are untouched.
// A true result means migration or asynchronous deletion still needs a retry.
func (r *BrokerReconciler) cleanupObsoleteBrokerResources(ctx context.Context, broker *mesheryv1alpha1.Broker) (bool, error) {
	if broker.DeletionTimestamp != nil || broker.UID == "" {
		return false, nil
	}
	obsolete, err := r.legacyBrokerChildren(ctx, broker)
	if err != nil || len(obsolete) == 0 {
		return false, err
	}
	ready, err := r.brokerMigrationReady(ctx, broker)
	if err != nil || !ready {
		return true, err
	}
	return r.deleteLegacyBrokerChildren(ctx, broker, obsolete)
}

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
		if m.Spec.Broker.Native.Name != broker.Name || m.Spec.Broker.Native.Namespace != broker.Namespace {
			continue
		}
		ready, err := r.meshSyncBrokerRolledOut(ctx, broker, m)
		if err != nil || !ready {
			return false, err
		}
	}
	return true, nil
}

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

func replacementBrokerRolloutComplete(broker *mesheryv1alpha1.Broker, current *appsv1.StatefulSet, replicas int32) bool {
	return isControlledByBroker(broker, current) && current.DeletionTimestamp == nil &&
		current.Status.ObservedGeneration >= current.Generation && current.Status.CurrentRevision != "" &&
		current.Status.CurrentRevision == current.Status.UpdateRevision &&
		current.Status.Replicas == replicas && current.Status.UpdatedReplicas == replicas && current.Status.ReadyReplicas == replicas
}

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

func meshSyncDeploymentRolloutComplete(deployment *appsv1.Deployment, replicas int32) bool {
	return deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == replicas &&
		deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.UpdatedReplicas == replicas && deployment.Status.Replicas == replicas &&
		deployment.Status.ReadyReplicas == replicas && deployment.Status.AvailableReplicas == replicas &&
		(deployment.Status.TerminatingReplicas == nil || *deployment.Status.TerminatingReplicas == 0)
}

func brokerEnvMatches(want corev1.Container, containers []corev1.Container) bool {
	for _, current := range containers {
		if current.Name != want.Name {
			continue
		}
		for _, expected := range want.Env {
			if expected.Name != "BROKER_URL" && expected.Name != "NATS_TOKEN" {
				continue
			}
			found := false
			for _, actual := range current.Env {
				if actual.Name == expected.Name && equality.Semantic.DeepEqual(actual, expected) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	return false
}
