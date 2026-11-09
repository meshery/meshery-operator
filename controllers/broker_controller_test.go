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
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/meshery/meshery-operator/api/v1alpha1"
	brokerpackage "github.com/meshery/meshery-operator/pkg/broker"
	meshsyncpackage "github.com/meshery/meshery-operator/pkg/meshsync"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testOldBrokerURL = "nats://old:4222"
	testNatsTokenEnv = "NATS_TOKEN"
	testBrokerURLEnv = "BROKER_URL"
	defaultNamespace = "default"
	testAPIVersion   = "meshery.io/v1alpha1"
	testBrokerKind   = "Broker"
	testMeshSyncKind = "MeshSync"
	// natsObjectName is the fixed chart-release name of the NATS StatefulSet
	// and client Service, regardless of the Broker CR's name.
	natsObjectName   = "meshery-nats"
	appLabelKey      = "app"
	brokerLabelValue = "broker"
)

var _ = Describe("The test cases for customize resource: Broker's controller ", func() {

	var (
		namespace string
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("Testing Broker's nothing found logic", func() {
		It("Getting broker resource should be failing", func() {
			namespace = defaultNamespace
			broker := &v1alpha1.Broker{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, broker)
			Expect(err).To(HaveOccurred())
		})
	})

	It("Creating a broker resource", func() {
		namespace = defaultNamespace
		broker := &v1alpha1.Broker{
			TypeMeta: metav1.TypeMeta{
				APIVersion: testAPIVersion,
				Kind:       testBrokerKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaultNamespace,
				Namespace: namespace,
			},
			Spec: v1alpha1.BrokerSpec{
				Size: 1,
			}}

		Expect(k8sClient.Create(ctx, broker)).Should(Succeed())
	})

	It("Getting broker resource should be successful", func() {
		namespace = defaultNamespace
		broker := &v1alpha1.Broker{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, broker)
		Expect(err).ToNot(HaveOccurred())
		Expect(broker.Spec.Size).To(Equal(int32(1)))
	})

	It("Updating broker resource should be successful", func() {
		namespace = defaultNamespace

		Eventually(func() error {
			broker := &v1alpha1.Broker{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, broker)
			if err != nil {
				return err
			}
			broker.Spec.Size = 2
			return k8sClient.Update(ctx, broker)
		}, 2*time.Second, 100*time.Millisecond).Should(Succeed())

		By("Checking if the broker resource is updated")
		broker := &v1alpha1.Broker{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, broker)
		Expect(err).ToNot(HaveOccurred())
		Expect(broker.Spec.Size).To(Equal(int32(2)))
	})

	Context("Testing Broker's brokerpackage.CheckHealth function", func() {
		It("Checking broker health functions", func() {
			namespace = defaultNamespace
			broker := &v1alpha1.Broker{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, broker)
			Expect(err).ToNot(HaveOccurred())
			By("Ensuring no StatefulSet exists before health check")
			existingStatefulSet := &v1.StatefulSet{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: natsObjectName, Namespace: namespace}, existingStatefulSet)
			if err == nil {
				Expect(k8sClient.Delete(ctx, existingStatefulSet)).Should(Succeed())
				Eventually(func() bool {
					err = k8sClient.Get(ctx, types.NamespacedName{Name: natsObjectName, Namespace: namespace}, existingStatefulSet)
					return err != nil
				}).Should(BeTrue())
			}
			By("Checking if the broker is healthy, it should return an error")
			Expect(brokerpackage.CheckHealth(ctx, broker, k8sClient)).To(HaveOccurred())
			statefulSet := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      natsObjectName,
				},
				Spec: v1.StatefulSetSpec{
					Replicas: &broker.Spec.Size,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							appLabelKey: brokerLabelValue,
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: defaultNamespace,
							Labels: map[string]string{
								appLabelKey: brokerLabelValue,
							},
						},
						Spec: corev1.PodSpec{
							// A pod template must declare at least one container;
							// the apiserver rejects an empty container list with a
							// 422 (spec.template.spec.containers: Required value).
							Containers: []corev1.Container{
								{
									Name:  "nats",
									Image: "nats:latest",
								},
							},
						},
					},
				},
			}

			By("Creating statefulset resources for testing broker")
			// The BrokerReconciler runs against this same "default" Broker CR and
			// server-side-applies the meshery-nats StatefulSet on every reconcile,
			// so it can recreate the object in the window between the delete above
			// and this create. Adopt the existing StatefulSet on AlreadyExists
			// rather than racing the controller (previously a flaky 409 here).
			err = k8sClient.Create(ctx, statefulSet)
			if apierrors.IsAlreadyExists(err) {
				err = k8sClient.Get(ctx, types.NamespacedName{Name: natsObjectName, Namespace: namespace}, statefulSet)
			}
			Expect(err).ToNot(HaveOccurred())

			By("Health must still fail until ReadyReplicas reaches the desired count")
			Expect(brokerpackage.CheckHealth(ctx, broker, k8sClient)).To(HaveOccurred())

			By("Driving the StatefulSet to ready via the status subresource (no kubelet)")
			// Re-fetch before each attempt: a concurrent controller reconcile can
			// bump the StatefulSet's resourceVersion and make a one-shot status
			// update conflict.
			Eventually(func() error {
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: natsObjectName, Namespace: namespace}, statefulSet); err != nil {
					return err
				}
				statefulSet.Status.Replicas = broker.Spec.Size
				statefulSet.Status.ReadyReplicas = broker.Spec.Size
				return k8sClient.Status().Update(ctx, statefulSet)
			}, 2*time.Second, 100*time.Millisecond).Should(Succeed())

			By("Checking if the broker is healthy, it should be successful")
			Expect(brokerpackage.CheckHealth(ctx, broker, k8sClient)).To(Succeed())
		})

	})

	Context("Testing Broker's Cleanup logic", func() {
		It("Deleting broker resource should be succeeding", func() {
			namespace = defaultNamespace
			broker := &v1alpha1.Broker{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, broker)
			Expect(err).ToNot(HaveOccurred())
			Expect(k8sClient.Delete(ctx, broker)).Should(Succeed())
		})
	})
})

type brokerCleanupFixture struct {
	broker  *v1alpha1.Broker
	objects []client.Object
}

func newBrokerCleanupFixture() *brokerCleanupFixture {
	one := int32(1)
	b := &v1alpha1.Broker{ObjectMeta: metav1.ObjectMeta{Name: "custom-broker", Namespace: "migration", UID: "broker-uid"}, Spec: v1alpha1.BrokerSpec{Size: one}}
	b.Status.Endpoint.Internal = "meshery-nats.migration.svc.cluster.local:4222"
	owner := *metav1.NewControllerRef(b, v1alpha1.GroupVersion.WithKind("Broker"))
	f := &brokerCleanupFixture{broker: b, objects: []client.Object{b}}
	for _, object := range brokerpackage.GetObjects(b) {
		o := object.(client.Object)
		o.SetUID(types.UID("current-" + o.GetName()))
		o.SetGeneration(1)
		o.SetOwnerReferences([]metav1.OwnerReference{owner})
		if sts, ok := o.(*v1.StatefulSet); ok {
			sts.Status = v1.StatefulSetStatus{ObservedGeneration: 1, Replicas: one, ReadyReplicas: one, UpdatedReplicas: one, CurrentRevision: "new", UpdateRevision: "new"}
		}
		f.objects = append(f.objects, o)
	}
	f.objects = append(f.objects,
		&v1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: b.Name, Namespace: b.Namespace, UID: "legacy-sts", Labels: legacyBrokerTestLabels(), OwnerReferences: []metav1.OwnerReference{owner}}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: b.Name, Namespace: b.Namespace, UID: "legacy-service", Labels: legacyBrokerTestLabels(), OwnerReferences: []metav1.OwnerReference{owner}}},
		brokerpackage.BuildAuthSecret(b.Namespace, "test-token"),
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "keep-data", Namespace: b.Namespace, Labels: legacyBrokerTestLabels(), OwnerReferences: []metav1.OwnerReference{owner}}},
	)
	return f
}

func (f *brokerCleanupFixture) reconciler(t *testing.T) *BrokerReconciler {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := k8sscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&v1.StatefulSet{}, &v1.Deployment{}, &v1alpha1.Broker{}).WithObjects(f.objects...).Build()
	return &BrokerReconciler{Client: c, Scheme: scheme, Log: logr.Discard(), Recorder: record.NewFakeRecorder(10)}
}

func (f *brokerCleanupFixture) addMeshSync() *v1.Deployment {
	m := &v1alpha1.MeshSync{ObjectMeta: metav1.ObjectMeta{Name: "consumer", Namespace: "other-namespace", UID: "meshsync-uid"}, Spec: v1alpha1.MeshSyncSpec{Size: 1, Broker: v1alpha1.MeshsyncBroker{Native: v1alpha1.NativeMeshsyncBroker{Name: f.broker.Name, Namespace: f.broker.Namespace}}}}
	m.Status.PublishingTo = natsURL(f.broker.Status.Endpoint.Internal)
	d := meshsyncpackage.GetServerObject(m, brokerpackage.AuthSecretName).(*v1.Deployment)
	d.Generation = 2
	d.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(m, v1alpha1.GroupVersion.WithKind(testMeshSyncKind))}
	d.Status = v1.DeploymentStatus{ObservedGeneration: 2, Replicas: 1, UpdatedReplicas: 1, ReadyReplicas: 1, AvailableReplicas: 1}
	f.objects = append(f.objects, m, d)
	return d
}

func (f *brokerCleanupFixture) addMeshSyncSameNamespaceDefaulted() *v1.Deployment {
	m := &v1alpha1.MeshSync{ObjectMeta: metav1.ObjectMeta{Name: "consumer-same-ns", Namespace: f.broker.Namespace, UID: "meshsync-same-ns-uid"}, Spec: v1alpha1.MeshSyncSpec{Size: 1, Broker: v1alpha1.MeshsyncBroker{Native: v1alpha1.NativeMeshsyncBroker{Name: f.broker.Name}}}}
	m.Status.PublishingTo = natsURL(f.broker.Status.Endpoint.Internal)
	d := meshsyncpackage.GetServerObject(m, brokerpackage.AuthSecretName).(*v1.Deployment)
	d.Generation = 2
	d.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(m, v1alpha1.GroupVersion.WithKind(testMeshSyncKind))}
	d.Status = v1.DeploymentStatus{ObservedGeneration: 2, Replicas: 1, UpdatedReplicas: 1, ReadyReplicas: 1, AvailableReplicas: 1}
	f.objects = append(f.objects, m, d)
	return d
}

func (f *brokerCleanupFixture) addMeshSyncNamespaceOnly() *v1.Deployment {
	m := &v1alpha1.MeshSync{ObjectMeta: metav1.ObjectMeta{Name: "consumer-ns-only", Namespace: f.broker.Namespace, UID: "meshsync-ns-only-uid"}, Spec: v1alpha1.MeshSyncSpec{Size: 1, Broker: v1alpha1.MeshsyncBroker{Native: v1alpha1.NativeMeshsyncBroker{Namespace: f.broker.Namespace}}}}
	m.Status.PublishingTo = natsURL(f.broker.Status.Endpoint.Internal)
	d := meshsyncpackage.GetServerObject(m, brokerpackage.AuthSecretName).(*v1.Deployment)
	d.Generation = 2
	d.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(m, v1alpha1.GroupVersion.WithKind(testMeshSyncKind))}
	d.Status = v1.DeploymentStatus{ObservedGeneration: 2, Replicas: 1, UpdatedReplicas: 1, ReadyReplicas: 1, AvailableReplicas: 1}
	f.objects = append(f.objects, m, d)
	return d
}

func TestIsLegacyBrokerChild(t *testing.T) {
	tests := []struct {
		modify func(*v1alpha1.Broker, client.Object)
		name   string
		want   bool
	}{
		{name: "custom Broker name", modify: func(_ *v1alpha1.Broker, _ client.Object) {}, want: true},
		{name: "hub owner version", modify: func(_ *v1alpha1.Broker, o client.Object) {
			o.GetOwnerReferences()[0].APIVersion = "meshery.io/v1alpha2"
		}, want: true},
		{name: "no owner", modify: func(_ *v1alpha1.Broker, o client.Object) { o.SetOwnerReferences(nil) }, want: false},
		{name: "different UID", modify: func(_ *v1alpha1.Broker, o client.Object) { o.GetOwnerReferences()[0].UID = "other" }, want: false},
		{name: "noncontrolling reference", modify: func(_ *v1alpha1.Broker, o client.Object) {
			value := false
			o.GetOwnerReferences()[0].Controller = &value
		}, want: false},
		{name: "wrong owner group", modify: func(_ *v1alpha1.Broker, o client.Object) { o.GetOwnerReferences()[0].APIVersion = "other.io/v1alpha1" }, want: false},
		{name: "wrong owner kind", modify: func(_ *v1alpha1.Broker, o client.Object) { o.GetOwnerReferences()[0].Kind = testMeshSyncKind }, want: false},
		{name: "wrong owner name", modify: func(_ *v1alpha1.Broker, o client.Object) { o.GetOwnerReferences()[0].Name = "other" }, want: false},
		{name: "missing legacy label", modify: func(_ *v1alpha1.Broker, o client.Object) { delete(o.GetLabels(), "component") }, want: false},
		{name: "unverified legacy name", modify: func(_ *v1alpha1.Broker, o client.Object) { o.SetName("other") }, want: false},
		{name: "different namespace", modify: func(_ *v1alpha1.Broker, o client.Object) { o.SetNamespace("other") }, want: false},
		{name: "current desired identity", modify: func(b *v1alpha1.Broker, o client.Object) {
			b.Name = natsObjectName
			o.SetName(b.Name)
			o.GetOwnerReferences()[0].Name = b.Name
		}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, kind := range []string{"StatefulSet", "Service"} {
				f := newBrokerCleanupFixture()
				var o client.Object
				for _, candidate := range f.objects {
					if candidate.GetName() == f.broker.Name && brokerChildKind(candidate) == kind {
						o = candidate
						break
					}
				}
				test.modify(f.broker, o)
				if got := isLegacyBrokerChild(f.broker, o); got != test.want {
					t.Fatalf("%T: match=%v, want %v", o, got, test.want)
				}
			}
		})
	}
}

func TestBrokerCleanupMigrationGate(t *testing.T) {
	tests := []struct {
		modify  func(*brokerCleanupFixture)
		name    string
		pending bool
	}{
		{name: "replacement ready", modify: func(_ *brokerCleanupFixture) {}, pending: false},
		{name: "missing endpoint", modify: func(f *brokerCleanupFixture) { f.broker.Status.Endpoint.Internal = "" }, pending: true},
		{name: "replacement unready", modify: func(f *brokerCleanupFixture) {
			f.replacementStatefulSet().Status.ReadyReplicas = 0
		}, pending: true},
		{name: "replacement revision outdated", modify: func(f *brokerCleanupFixture) {
			f.replacementStatefulSet().Status.CurrentRevision = "old"
		}, pending: true},
		{name: "replacement generation unobserved", modify: func(f *brokerCleanupFixture) {
			f.replacementStatefulSet().Status.ObservedGeneration = 0
		}, pending: true},
		{name: "cross namespace consumer ready", modify: func(f *brokerCleanupFixture) { f.addMeshSync() }, pending: false},
		{name: "same namespace consumer defaulted ready", modify: func(f *brokerCleanupFixture) { f.addMeshSyncSameNamespaceDefaulted() }, pending: false},
		{name: "same namespace consumer defaulted pending rollout", modify: func(f *brokerCleanupFixture) { f.addMeshSyncSameNamespaceDefaulted().Status.UpdatedReplicas = 0 }, pending: true},
		{name: "namespace-only consumer pending rollout", modify: func(f *brokerCleanupFixture) { f.addMeshSyncNamespaceOnly().Status.UpdatedReplicas = 0 }, pending: true},
		{name: "consumer rollout pending", modify: func(f *brokerCleanupFixture) { f.addMeshSync().Status.UpdatedReplicas = 0 }, pending: true},
		{name: "consumer generation unobserved", modify: func(f *brokerCleanupFixture) { f.addMeshSync().Status.ObservedGeneration = 1 }, pending: true},
		{name: "consumer unavailable", modify: func(f *brokerCleanupFixture) { f.addMeshSync().Status.AvailableReplicas = 0 }, pending: true},
		{name: "consumer terminating old pods", modify: func(f *brokerCleanupFixture) { value := int32(1); f.addMeshSync().Status.TerminatingReplicas = &value }, pending: true},
		{name: "consumer still uses old endpoint", modify: func(f *brokerCleanupFixture) {
			d := f.addMeshSync()
			for i := range d.Spec.Template.Spec.Containers[0].Env {
				e := &d.Spec.Template.Spec.Containers[0].Env[i]
				if e.Name == testBrokerURLEnv {
					e.Value = testOldBrokerURL
				}
			}
		}, pending: true},
		{name: "consumer lacks token reference", modify: func(f *brokerCleanupFixture) {
			d := f.addMeshSync()
			for i := range d.Spec.Template.Spec.Containers[0].Env {
				e := &d.Spec.Template.Spec.Containers[0].Env[i]
				if e.Name == testNatsTokenEnv {
					e.ValueFrom = nil
					e.Value = "literal"
				}
			}
		}, pending: true},
		{name: "missing consumer deployment", modify: func(f *brokerCleanupFixture) { f.addMeshSync(); f.objects = f.objects[:len(f.objects)-1] }, pending: true},
		{name: "Broker deletion path", modify: func(f *brokerCleanupFixture) {
			now := metav1.Now()
			f.broker.DeletionTimestamp = &now
			f.broker.Finalizers = []string{brokerFinalizer}
		}, pending: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newBrokerCleanupFixture()
			test.modify(f)
			r := f.reconciler(t)
			retryAfter, err := r.cleanupObsoleteBrokerResources(context.Background(), f.broker)
			if err != nil {
				t.Fatal(err)
			}
			preserve := test.pending || f.broker.DeletionTimestamp != nil
			for _, o := range []client.Object{&v1.StatefulSet{}, &corev1.Service{}} {
				err := r.Get(context.Background(), types.NamespacedName{Name: f.broker.Name, Namespace: f.broker.Namespace}, o)
				if preserve && err != nil {
					t.Fatalf("legacy %T must survive: %v", o, err)
				}
				if !preserve && !apierrors.IsNotFound(err) {
					t.Fatalf("legacy %T must be removed: %v", o, err)
				}
			}
			if test.pending && retryAfter == 0 {
				t.Fatal("pending migration must request a retry")
			}
		})
	}
}

func TestBrokerCleanupPreservationAndIdempotency(t *testing.T) {
	f := newBrokerCleanupFixture()
	r := f.reconciler(t)
	ctx := context.Background()
	if _, err := r.cleanupObsoleteBrokerResources(ctx, f.broker); err != nil {
		t.Fatal(err)
	}
	for _, object := range f.objects {
		if isLegacyBrokerChild(f.broker, object) {
			continue
		}
		if err := r.Get(ctx, client.ObjectKeyFromObject(object), object); err != nil {
			t.Fatalf("preserve %T %s: %v", object, object.GetName(), err)
		}
	}
	if retryAfter, err := r.cleanupObsoleteBrokerResources(ctx, f.broker); err != nil || retryAfter != 0 {
		t.Fatalf("repeat cleanup retry=%v, err=%v", retryAfter, err)
	}
	recorder := r.Recorder.(*record.FakeRecorder)
	if got := len(recorder.Events); got != 2 {
		t.Fatalf("cleanup events=%d, want exactly 2", got)
	}
}

func TestBrokerCleanupDeletePreconditions(t *testing.T) {
	f := newBrokerCleanupFixture()
	r := f.reconciler(t)
	ctx := context.Background()
	deletes := 0
	r.Client = interceptor.NewClient(r.Client.(client.WithWatch), interceptor.Funcs{Delete: func(ctx context.Context, c client.WithWatch, o client.Object, opts ...client.DeleteOption) error {
		deletes++
		options := &client.DeleteOptions{}
		for _, opt := range opts {
			opt.ApplyToDelete(options)
		}
		if options.Preconditions == nil || options.Preconditions.UID == nil || *options.Preconditions.UID != o.GetUID() || options.Preconditions.ResourceVersion == nil || *options.Preconditions.ResourceVersion != o.GetResourceVersion() {
			t.Fatal("delete must guard both UID and resourceVersion")
		}
		return c.Delete(ctx, o, opts...)
	}})
	if _, err := r.cleanupObsoleteBrokerResources(ctx, f.broker); err != nil {
		t.Fatal(err)
	}
	if deletes != 2 {
		t.Fatalf("delete requests=%d, want 2", deletes)
	}
}

func TestBrokerCleanupDeleteFailures(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprintf("conflict=%v", conflict), func(t *testing.T) {
			f := newBrokerCleanupFixture()
			r := f.reconciler(t)
			r.Client = interceptor.NewClient(r.Client.(client.WithWatch), interceptor.Funcs{Delete: func(ctx context.Context, c client.WithWatch, o client.Object, _ ...client.DeleteOption) error {
				resource := schema.GroupResource{Resource: "statefulsets"}
				cause := apierrors.NewForbidden(resource, o.GetName(), nil)
				if conflict {
					return apierrors.NewConflict(resource, o.GetName(), cause)
				}
				return cause
			}})
			retryAfter, err := r.cleanupObsoleteBrokerResources(context.Background(), f.broker)
			if retryAfter != 5*time.Second {
				t.Fatal("failed deletion needs a fast retry")
			}
			if conflict && err != nil {
				t.Fatalf("precondition conflict should revalidate: %v", err)
			}
			if !conflict && !apierrors.IsForbidden(err) {
				t.Fatalf("permission error must be surfaced: %v", err)
			}
			if len(r.Recorder.(*record.FakeRecorder).Events) != 0 {
				t.Fatal("failed deletion must not emit a success event")
			}
		})
	}
}

func (f *brokerCleanupFixture) replacementStatefulSet() *v1.StatefulSet {
	for _, object := range f.objects {
		if sts, ok := object.(*v1.StatefulSet); ok && sts.Name == natsObjectName {
			return sts
		}
	}
	panic("replacement StatefulSet missing from fixture")
}

func legacyBrokerTestLabels() map[string]string {
	return map[string]string{appLabelKey: "meshery", "component": brokerLabelValue}
}

// TestBrokerCleanupRetryIntervals distinguishes migration polling from deletion confirmation.
func TestBrokerCleanupRetryIntervals(t *testing.T) {
	markDeleting := func(f *brokerCleanupFixture, all bool) {
		for _, o := range f.objects {
			if isLegacyBrokerChild(f.broker, o) && (all || brokerChildKind(o) == "StatefulSet") {
				now := metav1.Now()
				o.SetDeletionTimestamp(&now)
				o.SetFinalizers([]string{"test.example/preserve-until-confirmed"})
			}
		}
	}
	tests := []struct {
		prepare    func(*brokerCleanupFixture)
		name       string
		want       time.Duration
		wantEvents int
	}{
		{name: "replacement not ready", prepare: func(f *brokerCleanupFixture) {
			f.replacementStatefulSet().Status.ReadyReplicas = 0
		}, want: time.Minute},
		{name: "consumer rollout blocked", prepare: func(f *brokerCleanupFixture) {
			f.addMeshSync().Status.UpdatedReplicas = 0
		}, want: time.Minute},
		{name: "deletion requested", prepare: func(_ *brokerCleanupFixture) {}, want: 5 * time.Second, wantEvents: 2},
		{name: "deletion already pending", prepare: func(f *brokerCleanupFixture) {
			markDeleting(f, true)
		}, want: 5 * time.Second},
		{name: "readiness drops after deletion started", prepare: func(f *brokerCleanupFixture) {
			markDeleting(f, true)
			f.replacementStatefulSet().Status.ReadyReplicas = 0
		}, want: 5 * time.Second},
		{name: "partial deletion with blocked migration", prepare: func(f *brokerCleanupFixture) {
			markDeleting(f, false)
			f.addMeshSync().Status.UpdatedReplicas = 0
		}, want: 5 * time.Second},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newBrokerCleanupFixture()
			test.prepare(f)
			r := f.reconciler(t)
			result, err := r.reconcileBrokerCleanup(context.Background(), f.broker)
			if err != nil || result.RequeueAfter != test.want {
				t.Fatalf("retry=%v, err=%v; want %v", result.RequeueAfter, err, test.want)
			}
			if got := len(r.Recorder.(*record.FakeRecorder).Events); got != test.wantEvents {
				t.Fatalf("events=%d, want %d", got, test.wantEvents)
			}
			if test.wantEvents == 0 {
				for _, o := range []client.Object{&v1.StatefulSet{}, &corev1.Service{}} {
					if getErr := r.Get(context.Background(), client.ObjectKeyFromObject(f.broker), o); getErr != nil {
						t.Fatalf("pending legacy %T must be retained: %v", o, getErr)
					}
				}
			}
			if test.wantEvents == 2 {
				result, err = r.reconcileBrokerCleanup(context.Background(), f.broker)
				if err != nil || result.RequeueAfter != 0 {
					t.Fatalf("completed cleanup retry=%v, err=%v", result.RequeueAfter, err)
				}
			}
		})
	}
}

// TestBrokerCleanupEquivalentSecretReference exercises an equivalent stored Secret selector.
func TestBrokerCleanupEquivalentSecretReference(t *testing.T) {
	f := newBrokerCleanupFixture()
	deployment := f.addMeshSync()
	for i := range deployment.Spec.Template.Spec.Containers[0].Env {
		e := &deployment.Spec.Template.Spec.Containers[0].Env[i]
		if e.Name == testNatsTokenEnv {
			required := false
			e.ValueFrom.SecretKeyRef.Optional = &required
		}
	}
	r := f.reconciler(t)
	if _, err := r.cleanupObsoleteBrokerResources(context.Background(), f.broker); err != nil {
		t.Fatal(err)
	}
	for _, o := range []client.Object{&v1.StatefulSet{}, &corev1.Service{}} {
		err := r.Get(context.Background(), client.ObjectKeyFromObject(f.broker), o)
		if !apierrors.IsNotFound(err) {
			t.Fatalf("equivalent required Secret reference must allow removal of legacy %T: %v", o, err)
		}
	}
}

// TestBrokerCleanupEnvironmentSemantics preserves endpoint and authentication requirements.
func TestBrokerCleanupEnvironmentSemantics(t *testing.T) {
	f := newBrokerCleanupFixture()
	want := f.addMeshSync().Spec.Template.Spec.Containers[0]
	tests := []struct {
		modify  func(*corev1.Container)
		name    string
		matches bool
	}{
		{name: "exact configuration", modify: func(_ *corev1.Container) {}, matches: true},
		{name: "explicitly required Secret", modify: func(c *corev1.Container) {
			value := false
			brokerCleanupTestEnv(c, testNatsTokenEnv).ValueFrom.SecretKeyRef.Optional = &value
		}, matches: true},
		{name: "optional Secret", modify: func(c *corev1.Container) {
			value := true
			brokerCleanupTestEnv(c, testNatsTokenEnv).ValueFrom.SecretKeyRef.Optional = &value
		}},
		{name: "wrong Secret name", modify: func(c *corev1.Container) {
			brokerCleanupTestEnv(c, testNatsTokenEnv).ValueFrom.SecretKeyRef.Name = "other-secret"
		}},
		{name: "wrong Secret key", modify: func(c *corev1.Container) {
			brokerCleanupTestEnv(c, testNatsTokenEnv).ValueFrom.SecretKeyRef.Key = "other-key"
		}},
		{name: "literal token", modify: func(c *corev1.Container) {
			e := brokerCleanupTestEnv(c, testNatsTokenEnv)
			e.ValueFrom, e.Value = nil, "literal-token"
		}},
		{name: "different token source", modify: func(c *corev1.Container) {
			brokerCleanupTestEnv(c, testNatsTokenEnv).ValueFrom = &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}
		}},
		{name: "old broker URL", modify: func(c *corev1.Container) {
			brokerCleanupTestEnv(c, testBrokerURLEnv).Value = testOldBrokerURL
		}},
		{name: "indirect broker URL", modify: func(c *corev1.Container) {
			e := brokerCleanupTestEnv(c, testBrokerURLEnv)
			e.Value = ""
			e.ValueFrom = &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "broker"}, Key: "url"}}
		}},
		{name: "missing token", modify: func(c *corev1.Container) {
			for i := range c.Env {
				if c.Env[i].Name == testNatsTokenEnv {
					c.Env = append(c.Env[:i], c.Env[i+1:]...)
					return
				}
			}
		}},
		{name: "missing URL", modify: func(c *corev1.Container) {
			for i := range c.Env {
				if c.Env[i].Name == testBrokerURLEnv {
					c.Env = append(c.Env[:i], c.Env[i+1:]...)
					return
				}
			}
		}},
		{name: "token defined after URL", modify: func(c *corev1.Container) {
			token, brokerURL := -1, -1
			for i := range c.Env {
				if c.Env[i].Name == testNatsTokenEnv {
					token = i
				}
				if c.Env[i].Name == testBrokerURLEnv {
					brokerURL = i
				}
			}
			c.Env[token], c.Env[brokerURL] = c.Env[brokerURL], c.Env[token]
		}},
		{name: "duplicate token", modify: func(c *corev1.Container) {
			duplicate := brokerCleanupTestEnv(c, testNatsTokenEnv).DeepCopy()
			duplicate.ValueFrom.SecretKeyRef.Name = "other-secret"
			c.Env = append(c.Env, *duplicate)
		}},
		{name: "duplicate URL", modify: func(c *corev1.Container) {
			c.Env = append(c.Env, corev1.EnvVar{Name: testBrokerURLEnv, Value: testOldBrokerURL})
		}},
		{name: "unrelated env addition", modify: func(c *corev1.Container) {
			c.Env = append(c.Env, corev1.EnvVar{Name: "EXTRA_ENV", Value: "unrelated"})
		}, matches: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := want.DeepCopy()
			test.modify(current)
			if got := brokerEnvMatches(want, []corev1.Container{*current}); got != test.matches {
				t.Fatalf("match=%v, want %v", got, test.matches)
			}
		})
	}
}

func brokerCleanupTestEnv(container *corev1.Container, name string) *corev1.EnvVar {
	for i := range container.Env {
		if container.Env[i].Name == name {
			return &container.Env[i]
		}
	}
	panic("broker environment fixture missing " + name)
}

// TestBrokerCleanupFailureRecovery verifies failure reporting and eventual condition recovery.
func TestBrokerCleanupFailureRecovery(t *testing.T) {
	ctx := context.Background()
	f := newBrokerCleanupFixture()
	r := f.reconciler(t)
	original := r.Client
	r.Client = interceptor.NewClient(original.(client.WithWatch), interceptor.Funcs{Delete: func(_ context.Context, _ client.WithWatch, o client.Object, _ ...client.DeleteOption) error {
		return apierrors.NewForbidden(schema.GroupResource{Resource: "statefulsets"}, o.GetName(), nil)
	}})
	current := &v1alpha1.Broker{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(f.broker), current); err != nil {
		t.Fatal(err)
	}
	result, err := r.reconcileBrokerCleanup(ctx, current)
	if err == nil || result.RequeueAfter != 0 {
		t.Fatalf("API errors must be returned for controller backoff; retry=%v err=%v", result.RequeueAfter, err)
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(f.broker), current); err != nil {
		t.Fatal(err)
	}
	condition := meta.FindStatusCondition(current.Status.Conditions, "Cleanup")
	if condition == nil || condition.Status != metav1.ConditionFalse {
		t.Fatal("deletion failure must record a false Cleanup condition")
	}
	if len(r.Recorder.(*record.FakeRecorder).Events) != 0 {
		t.Fatal("failed deletion must not record success Events")
	}
	r.Client = original
	if _, err := r.reconcileBrokerCleanup(ctx, current); err != nil {
		t.Fatal(err)
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(f.broker), current); err != nil {
		t.Fatal(err)
	}
	if result, err := r.reconcileBrokerCleanup(ctx, current); err != nil || result.RequeueAfter != 0 {
		t.Fatalf("successful cleanup must stop retrying: result=%v err=%v", result, err)
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(f.broker), current); err != nil {
		t.Fatal(err)
	}
	condition = meta.FindStatusCondition(current.Status.Conditions, "Cleanup")
	if condition == nil || condition.Status != metav1.ConditionTrue {
		t.Fatal("successful cleanup must clear the failure condition")
	}
}
