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
	"slices"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/meshery/meshery-operator/api/v1alpha1"
	mesheryv1alpha2 "github.com/meshery/meshery-operator/api/v1alpha2"
	brokerpackage "github.com/meshery/meshery-operator/pkg/broker"
	meshkiterrors "github.com/meshery/meshkit/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("The test cases for customize resource: MeshSync's controller ", func() {

	var (
		namespace string
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("Testing meshSync's nothing found logic", func() {
		It("Getting meshSync resource should be failing", func() {
			namespace = defaultNamespace
			meshSync := &v1alpha1.MeshSync{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, meshSync)
			Expect(err).To(HaveOccurred())
		})
	})

	It("Creating a meshSync resource", func() {
		namespace = defaultNamespace
		meshSync := &v1alpha1.MeshSync{
			TypeMeta: metav1.TypeMeta{
				APIVersion: testAPIVersion,
				Kind:       testMeshSyncKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaultNamespace,
				Namespace: namespace,
			},
			Spec: v1alpha1.MeshSyncSpec{
				Size: 1,
			}}

		Expect(k8sClient.Create(ctx, meshSync)).Should(Succeed())
	})

	It("Getting meshSync resource should be successful", func() {
		namespace = defaultNamespace
		meshSync := &v1alpha1.MeshSync{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, meshSync)
		Expect(err).ToNot(HaveOccurred())
		Expect(meshSync.Spec.Size).To(Equal(int32(1)))
	})

	It("Updating meshSync resource should be successful", func() {
		namespace = defaultNamespace
		Eventually(func() error {
			meshSync := &v1alpha1.MeshSync{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, meshSync)
			if err != nil {
				return err
			}
			meshSync.Spec.Size = 2
			return k8sClient.Update(ctx, meshSync)
		}, "10s", "100ms").Should(Succeed())

		By("Checking if the meshSync resource is updated")
		meshSync := &v1alpha1.MeshSync{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, meshSync)
		Expect(err).ToNot(HaveOccurred())
		Expect(meshSync.Spec.Size).To(Equal(int32(2)))
	})

	Context("Testing MeshSync's Cleanup logic", func() {
		It("Deleting meshSync resource should be succeeding", func() {
			namespace = defaultNamespace
			meshSync := &v1alpha1.MeshSync{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: defaultNamespace, Namespace: namespace}, meshSync)
			Expect(err).ToNot(HaveOccurred())
			Expect(k8sClient.Delete(ctx, meshSync)).Should(Succeed())
		})
	})

})

func setupMeshSyncTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := k8sscheme.AddToScheme(s); err != nil {
		t.Fatalf("failed to add k8s scheme: %v", err)
	}
	if err := v1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("failed to add v1alpha1 scheme: %v", err)
	}
	if err := mesheryv1alpha2.AddToScheme(s); err != nil {
		t.Fatalf("failed to add v1alpha2 scheme: %v", err)
	}
	return s
}

func TestMeshSyncReconcileBrokerConfig(t *testing.T) {
	scheme := setupMeshSyncTestScheme(t)

	t.Run("SuccessWithExplicitNamespace", func(t *testing.T) {
		broker := &v1alpha1.Broker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBrokerName,
				Namespace: testMesheryNamespace,
			},
		}
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      natsObjectName,
				Namespace: testMesheryNamespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: testNatsClusterIP,
				Ports: []corev1.ServicePort{
					{Name: testNatsClientPort, Port: 4222},
				},
			},
		}
		meshsync := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testMeshSyncName,
				Namespace: testMesheryNamespace,
			},
			Spec: v1alpha1.MeshSyncSpec{
				Broker: v1alpha1.MeshsyncBroker{
					Native: v1alpha1.NativeMeshsyncBroker{
						Name:      testBrokerName,
						Namespace: testMesheryNamespace,
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(broker, service, meshsync).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard(), KubeConfig: &rest.Config{}}

		err := r.reconcileBrokerConfig(context.Background(), meshsync)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedURL := testNatsURL
		if meshsync.Status.PublishingTo != expectedURL {
			t.Errorf("PublishingTo = %q, want %q", meshsync.Status.PublishingTo, expectedURL)
		}
	})

	t.Run("SuccessWithDefaultedNamespace", func(t *testing.T) {
		broker := &v1alpha1.Broker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBrokerName,
				Namespace: testMesheryNamespace,
			},
		}
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      natsObjectName,
				Namespace: testMesheryNamespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: testNatsClusterIP,
				Ports: []corev1.ServicePort{
					{Name: testNatsClientPort, Port: 4222},
				},
			},
		}
		// Notice: native.Namespace is omitted ("")
		meshsync := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testMeshSyncName,
				Namespace: testMesheryNamespace,
			},
			Spec: v1alpha1.MeshSyncSpec{
				Broker: v1alpha1.MeshsyncBroker{
					Native: v1alpha1.NativeMeshsyncBroker{
						Name: testBrokerName,
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(broker, service, meshsync).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard(), KubeConfig: &rest.Config{}}

		err := r.reconcileBrokerConfig(context.Background(), meshsync)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedURL := testNatsURL
		if meshsync.Status.PublishingTo != expectedURL {
			t.Errorf("PublishingTo = %q, want %q", meshsync.Status.PublishingTo, expectedURL)
		}
	})

	t.Run("SuccessWithNamespaceOnlyReference", func(t *testing.T) {
		// When spec.broker.native.name is omitted but namespace is specified,
		// the controller skips fetching a named Broker CR and derives the endpoint
		// directly from the meshery-nats Service in that namespace.
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      natsObjectName,
				Namespace: "custom-broker-ns",
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.96.0.25",
				Ports: []corev1.ServicePort{
					{Name: testNatsClientPort, Port: 4222},
				},
			},
		}
		meshsync := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testMeshSyncName,
				Namespace: testMesheryNamespace,
			},
			Spec: v1alpha1.MeshSyncSpec{
				Broker: v1alpha1.MeshsyncBroker{
					Native: v1alpha1.NativeMeshsyncBroker{
						Namespace: "custom-broker-ns",
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(service, meshsync).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard(), KubeConfig: &rest.Config{}}

		err := r.reconcileBrokerConfig(context.Background(), meshsync)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedURL := "nats://10.96.0.25:4222"
		if meshsync.Status.PublishingTo != expectedURL {
			t.Errorf("PublishingTo = %q, want %q", meshsync.Status.PublishingTo, expectedURL)
		}
	})
}

// TestMeshSyncReconcileBrokerConfigOverrides covers the override, not-found,
// and custom-URL paths. It is split from TestMeshSyncReconcileBrokerConfig so
// neither function exceeds the cyclop complexity limit.
func TestMeshSyncReconcileBrokerConfigOverrides(t *testing.T) {
	scheme := setupMeshSyncTestScheme(t)

	t.Run("EndpointResolutionPreservesExternalEndpointOverride", func(t *testing.T) {
		broker := &v1alpha1.Broker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testOverrideBroker,
				Namespace: testMesheryNamespace,
			},
			Spec: v1alpha1.BrokerSpec{
				Service: v1alpha1.BrokerServiceSpec{
					ExternalEndpointOverride: "custom-nats.corp.internal:4222",
				},
			},
		}
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      natsObjectName,
				Namespace: testMesheryNamespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: testNatsClusterIP,
				Ports: []corev1.ServicePort{
					{Name: testNatsClientPort, Port: 4222},
				},
			},
		}
		meshsync := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testMeshSyncName,
				Namespace: testMesheryNamespace,
			},
			Spec: v1alpha1.MeshSyncSpec{
				Broker: v1alpha1.MeshsyncBroker{
					Native: v1alpha1.NativeMeshsyncBroker{
						Name:      testOverrideBroker,
						Namespace: testMesheryNamespace,
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(broker, service, meshsync).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard(), KubeConfig: &rest.Config{}}

		// Fetch the broker using the reconciler's lookup logic to verify the fetched struct retains spec
		fetched := &v1alpha1.Broker{}
		if err := r.Get(context.Background(), types.NamespacedName{Name: testOverrideBroker, Namespace: testMesheryNamespace}, fetched); err != nil {
			t.Fatalf("failed to fetch broker: %v", err)
		}
		if err := brokerpackage.GetEndpoint(context.Background(), fetched, c, r.KubeConfig.Host); err != nil {
			t.Fatalf("GetEndpoint failed: %v", err)
		}
		if fetched.Status.Endpoint.External != "custom-nats.corp.internal:4222" {
			t.Errorf("External endpoint override = %q, want %q", fetched.Status.Endpoint.External, "custom-nats.corp.internal:4222")
		}

		// Also verify reconcileBrokerConfig succeeds on this resource
		if err := r.reconcileBrokerConfig(context.Background(), meshsync); err != nil {
			t.Fatalf("reconcileBrokerConfig failed: %v", err)
		}
		if meshsync.Status.PublishingTo != testNatsURL {
			t.Errorf("PublishingTo = %q, want %q", meshsync.Status.PublishingTo, testNatsURL)
		}
	})

	t.Run("BrokerNotFoundReturnsErrGetBroker", func(t *testing.T) {
		// Even if a leftover meshery-nats service exists, the controller must
		// verify the Broker CR itself and return ErrGetBroker when absent.
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      natsObjectName,
				Namespace: testMesheryNamespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: testNatsClusterIP,
			},
		}
		meshsync := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testMeshSyncName,
				Namespace: testMesheryNamespace,
			},
			Spec: v1alpha1.MeshSyncSpec{
				Broker: v1alpha1.MeshsyncBroker{
					Native: v1alpha1.NativeMeshsyncBroker{
						Name:      "missing-broker",
						Namespace: testMesheryNamespace,
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(service, meshsync).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard(), KubeConfig: &rest.Config{}}

		err := r.reconcileBrokerConfig(context.Background(), meshsync)
		if err == nil {
			t.Fatal("expected error for non-existent broker, got nil")
		}
		if code := meshkiterrors.GetCode(err); code != ErrGetBrokerCode {
			t.Errorf("error code = %q, want %q (err: %v)", code, ErrGetBrokerCode, err)
		}
	})

	t.Run("CustomBrokerURL", func(t *testing.T) {
		meshsync := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testMeshSyncName,
				Namespace: testMesheryNamespace,
			},
			Spec: v1alpha1.MeshSyncSpec{
				Broker: v1alpha1.MeshsyncBroker{
					Custom: v1alpha1.CustomMeshsyncBroker{
						URL: "nats://custom-endpoint:4222",
					},
				},
			},
		}

		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(meshsync).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard(), KubeConfig: &rest.Config{}}

		err := r.reconcileBrokerConfig(context.Background(), meshsync)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meshsync.Status.PublishingTo != "nats://custom-endpoint:4222" {
			t.Errorf("PublishingTo = %q, want %q", meshsync.Status.PublishingTo, "nats://custom-endpoint:4222")
		}
	})
}

func TestMeshSyncEnsureBrokerTokenSecret(t *testing.T) {
	scheme := setupMeshSyncTestScheme(t)

	t.Run("NoNativeBroker", func(t *testing.T) {
		m := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{Name: testSyncName, Namespace: testMesheryNamespace},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(m).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard()}

		secretName, err := r.ensureBrokerTokenSecret(context.Background(), m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if secretName != "" {
			t.Errorf("secretName = %q, want empty", secretName)
		}
	})

	t.Run("SameNamespaceExplicit", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      brokerpackage.AuthSecretName,
				Namespace: testMesheryNamespace,
			},
			Data: map[string][]byte{"token": []byte("test-token")},
		}
		m := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{Name: testSyncName, Namespace: testMesheryNamespace},
			Spec: v1alpha1.MeshSyncSpec{
				Broker: v1alpha1.MeshsyncBroker{
					Native: v1alpha1.NativeMeshsyncBroker{
						Name:      testBrokerName,
						Namespace: testMesheryNamespace,
					},
				},
			},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret, m).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard()}

		secretName, err := r.ensureBrokerTokenSecret(context.Background(), m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if secretName != brokerpackage.AuthSecretName {
			t.Errorf("secretName = %q, want %q", secretName, brokerpackage.AuthSecretName)
		}
	})

	t.Run("SameNamespaceDefaulted", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      brokerpackage.AuthSecretName,
				Namespace: testMesheryNamespace,
			},
			Data: map[string][]byte{"token": []byte("test-token")},
		}
		// Namespace omitted in native
		m := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{Name: testSyncName, Namespace: testMesheryNamespace},
			Spec: v1alpha1.MeshSyncSpec{
				Broker: v1alpha1.MeshsyncBroker{
					Native: v1alpha1.NativeMeshsyncBroker{
						Name: testBrokerName,
					},
				},
			},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret, m).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard()}

		secretName, err := r.ensureBrokerTokenSecret(context.Background(), m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if secretName != brokerpackage.AuthSecretName {
			t.Errorf("secretName = %q, want %q", secretName, brokerpackage.AuthSecretName)
		}
	})

	t.Run("TokenAuthNotInUse", func(t *testing.T) {
		m := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{Name: testSyncName, Namespace: testMesheryNamespace},
			Spec: v1alpha1.MeshSyncSpec{
				Broker: v1alpha1.MeshsyncBroker{
					Native: v1alpha1.NativeMeshsyncBroker{
						Name:      testBrokerName,
						Namespace: testMesheryNamespace,
					},
				},
			},
		}
		// Secret does not exist
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(m).Build()
		r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard()}

		secretName, err := r.ensureBrokerTokenSecret(context.Background(), m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if secretName != "" {
			t.Errorf("secretName = %q, want empty", secretName)
		}
	})
}

func TestMeshSyncMeshsyncsForBroker(t *testing.T) {
	scheme := setupMeshSyncTestScheme(t)

	// MeshSync 1: explicit namespace
	msExplicit := &v1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ms-explicit",
			Namespace: testMesheryNamespace,
		},
		Spec: v1alpha1.MeshSyncSpec{
			Broker: v1alpha1.MeshsyncBroker{
				Native: v1alpha1.NativeMeshsyncBroker{
					Name:      testBrokerName,
					Namespace: testMesheryNamespace,
				},
			},
		},
	}
	// MeshSync 2: omitted namespace (defaults to meshery)
	msDefaulted := &v1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ms-defaulted",
			Namespace: testMesheryNamespace,
		},
		Spec: v1alpha1.MeshSyncSpec{
			Broker: v1alpha1.MeshsyncBroker{
				Native: v1alpha1.NativeMeshsyncBroker{
					Name: testBrokerName,
				},
			},
		},
	}
	// MeshSync 3: explicit cross-namespace reference (in other-namespace referencing meshery/test-broker)
	msCrossNamespace := &v1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ms-cross-ns",
			Namespace: testOtherNamespace,
		},
		Spec: v1alpha1.MeshSyncSpec{
			Broker: v1alpha1.MeshsyncBroker{
				Native: v1alpha1.NativeMeshsyncBroker{
					Name:      testBrokerName,
					Namespace: testMesheryNamespace,
				},
			},
		},
	}
	// MeshSync 4: defaulted broker reference in another namespace (must NOT match meshery/test-broker)
	msOtherNsDefaulted := &v1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ms-other-ns-defaulted",
			Namespace: testOtherNamespace,
		},
		Spec: v1alpha1.MeshSyncSpec{
			Broker: v1alpha1.MeshsyncBroker{
				Native: v1alpha1.NativeMeshsyncBroker{
					Name: testBrokerName,
				},
			},
		},
	}
	// MeshSync 5: other broker name in same namespace
	msOther := &v1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ms-other",
			Namespace: testMesheryNamespace,
		},
		Spec: v1alpha1.MeshSyncSpec{
			Broker: v1alpha1.MeshsyncBroker{
				Native: v1alpha1.NativeMeshsyncBroker{
					Name:      "other-broker",
					Namespace: testMesheryNamespace,
				},
			},
		},
	}
	// MeshSync 6: namespace-only consumer in meshery (matches meshery/test-broker)
	msNamespaceOnly := &v1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ms-ns-only",
			Namespace: testMesheryNamespace,
		},
		Spec: v1alpha1.MeshSyncSpec{
			Broker: v1alpha1.MeshsyncBroker{
				Native: v1alpha1.NativeMeshsyncBroker{
					Namespace: testMesheryNamespace,
				},
			},
		},
	}
	// MeshSync 7: namespace-only consumer in other-namespace (must NOT match meshery/test-broker)
	msOtherNsNamespaceOnly := &v1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ms-other-ns-only",
			Namespace: testOtherNamespace,
		},
		Spec: v1alpha1.MeshSyncSpec{
			Broker: v1alpha1.MeshsyncBroker{
				Native: v1alpha1.NativeMeshsyncBroker{
					Namespace: testOtherNamespace,
				},
			},
		},
	}
	// MeshSync 8: consumer with no native broker configured (must NOT match meshery/test-broker)
	msNoNative := &v1alpha1.MeshSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ms-no-native",
			Namespace: testMesheryNamespace,
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(msExplicit, msDefaulted, msCrossNamespace, msOtherNsDefaulted, msOther, msNamespaceOnly, msOtherNsNamespaceOnly, msNoNative).Build()
	r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard()}

	t.Run("MatchesV1alpha1BrokerAndResolvesDefaultedNamespace", func(t *testing.T) {
		broker := &v1alpha1.Broker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBrokerName,
				Namespace: testMesheryNamespace,
			},
		}

		reqs := r.meshsyncsForBroker(context.Background(), broker)
		if len(reqs) != 4 {
			t.Fatalf("expected 4 reconcile requests, got %d: %+v", len(reqs), reqs)
		}
		matched := map[types.NamespacedName]bool{}
		for _, req := range reqs {
			matched[req.NamespacedName] = true
		}
		if !matched[types.NamespacedName{Name: "ms-explicit", Namespace: testMesheryNamespace}] {
			t.Errorf("missing request for ms-explicit in meshery")
		}
		if !matched[types.NamespacedName{Name: "ms-defaulted", Namespace: testMesheryNamespace}] {
			t.Errorf("missing request for ms-defaulted in meshery")
		}
		if !matched[types.NamespacedName{Name: "ms-cross-ns", Namespace: testOtherNamespace}] {
			t.Errorf("missing request for ms-cross-ns in other-namespace")
		}
		if !matched[types.NamespacedName{Name: "ms-ns-only", Namespace: testMesheryNamespace}] {
			t.Errorf("missing request for ms-ns-only in meshery")
		}
		if matched[types.NamespacedName{Name: "ms-other-ns-defaulted", Namespace: testOtherNamespace}] {
			t.Errorf("unexpected request for ms-other-ns-defaulted; namespace isolation was violated")
		}
		if matched[types.NamespacedName{Name: "ms-other-ns-only", Namespace: testOtherNamespace}] {
			t.Errorf("unexpected request for ms-other-ns-only; namespace isolation was violated")
		}
		if matched[types.NamespacedName{Name: "ms-no-native", Namespace: testMesheryNamespace}] {
			t.Errorf("unexpected request for ms-no-native; unconfigured native broker was matched")
		}
	})

	t.Run("MatchesV1alpha2StorageVersionBroker", func(t *testing.T) {
		brokerHub := &mesheryv1alpha2.Broker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testBrokerName,
				Namespace: testMesheryNamespace,
			},
		}

		reqs := r.meshsyncsForBroker(context.Background(), brokerHub)
		if len(reqs) != 4 {
			t.Fatalf("expected 4 reconcile requests for v1alpha2 broker, got %d", len(reqs))
		}
	})

	t.Run("NonMatchingBrokerReturnsEmpty", func(t *testing.T) {
		broker := &v1alpha1.Broker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "unreferenced-broker",
				Namespace: "unreferenced-namespace",
			},
		}

		reqs := r.meshsyncsForBroker(context.Background(), broker)
		if len(reqs) != 0 {
			t.Errorf("expected 0 reconcile requests, got %d", len(reqs))
		}
	})

	t.Run("NilAndEmptyReturnNil", func(t *testing.T) {
		if reqs := r.meshsyncsForBroker(context.Background(), nil); reqs != nil {
			t.Errorf("expected nil for nil obj, got %+v", reqs)
		}
		emptyBroker := &v1alpha1.Broker{}
		if reqs := r.meshsyncsForBroker(context.Background(), emptyBroker); reqs != nil {
			t.Errorf("expected nil for empty broker name, got %+v", reqs)
		}
	})
}

var _ = Describe("MeshSync lifecycle without finalizers", func() {
	ctx := context.Background()

	newMeshSync := func(namespace string, broker v1alpha1.MeshsyncBroker, finalizers ...string) *v1alpha1.MeshSync {
		meshsync := &v1alpha1.MeshSync{
			ObjectMeta: metav1.ObjectMeta{Name: testMeshSyncName, Namespace: namespace, Finalizers: finalizers},
			Spec:       v1alpha1.MeshSyncSpec{Size: 1, Broker: broker},
		}
		Expect(k8sClient.Create(ctx, meshsync)).To(Succeed())
		return meshsync
	}
	customBroker := v1alpha1.MeshsyncBroker{Custom: v1alpha1.CustomMeshsyncBroker{URL: "nats://custom-broker:4222"}}

	It("adds no finalizer, owns its Deployment and mirrored auth Secret, and deletes without operator involvement", func() {
		// A Broker in another namespace makes the reconcile mirror the NATS
		// auth Secret into the MeshSync's namespace - the second object this
		// controller creates.
		brokerNamespace := createTestNamespace(ctx, "meshsync-gc-broker")
		Expect(k8sClient.Create(ctx, &v1alpha1.Broker{
			ObjectMeta: metav1.ObjectMeta{Name: testBrokerName, Namespace: brokerNamespace},
			Spec:       v1alpha1.BrokerSpec{Size: 1},
		})).To(Succeed())
		meshsync := newMeshSync(createTestNamespace(ctx, "meshsync-gc"),
			v1alpha1.MeshsyncBroker{Native: v1alpha1.NativeMeshsyncBroker{Name: testBrokerName, Namespace: brokerNamespace}})
		key := client.ObjectKeyFromObject(meshsync)

		By("the Deployment and the mirrored Secret carry a controller ownerReference to the MeshSync")
		Eventually(func(g Gomega) {
			for _, current := range []client.Object{&appsv1.Deployment{}, &corev1.Secret{}} {
				name := meshsync.Name
				if _, ok := current.(*corev1.Secret); ok {
					name = brokerpackage.AuthSecretName
				}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: meshsync.Namespace}, current)).To(Succeed())
				expectControlledBy(g, current, meshsync, testMeshSyncKind)
			}
		}, 20*time.Second, 250*time.Millisecond).Should(Succeed())

		By("the reconciled MeshSync carries no finalizer")
		Consistently(func(g Gomega) {
			current := &v1alpha1.MeshSync{}
			g.Expect(k8sClient.Get(ctx, key, current)).To(Succeed())
			g.Expect(current.Finalizers).To(BeEmpty())
		}, time.Second, 100*time.Millisecond).Should(Succeed())

		By("deleting the MeshSync completes synchronously: nothing waits on the operator")
		Expect(k8sClient.Delete(ctx, meshsync)).To(Succeed())
		Expect(apierrors.IsNotFound(k8sClient.Get(ctx, key, &v1alpha1.MeshSync{}))).To(BeTrue())
	})

	It("strips the legacy finalizer from a live MeshSync and keeps reconciling it", func() {
		meshsync := newMeshSync(createTestNamespace(ctx, "meshsync-legacy"), customBroker, legacyMeshsyncFinalizer)
		Eventually(func(g Gomega) {
			current := &v1alpha1.MeshSync{}
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(meshsync), current)).To(Succeed())
			g.Expect(current.Finalizers).To(BeEmpty())
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(meshsync), &appsv1.Deployment{})).To(Succeed())
		}, 15*time.Second, 250*time.Millisecond).Should(Succeed())
	})

	It("strips the legacy finalizer from a Terminating MeshSync so its deletion can complete", func() {
		meshsync := newMeshSync(createTestNamespace(ctx, "meshsync-terminating"), customBroker, legacyMeshsyncFinalizer, testHoldFinalizer)
		key := client.ObjectKeyFromObject(meshsync)
		Expect(k8sClient.Delete(ctx, meshsync)).To(Succeed())
		Eventually(func(g Gomega) {
			current := &v1alpha1.MeshSync{}
			g.Expect(k8sClient.Get(ctx, key, current)).To(Succeed())
			g.Expect(current.DeletionTimestamp).ToNot(BeNil())
			g.Expect(current.Finalizers).To(ConsistOf(testHoldFinalizer))
		}, 15*time.Second, 250*time.Millisecond).Should(Succeed())

		By("releasing the other writer's finalizer, the MeshSync is removed")
		Eventually(func() error {
			current := &v1alpha1.MeshSync{}
			if err := k8sClient.Get(ctx, key, current); err != nil {
				return err
			}
			current.Finalizers = nil
			return k8sClient.Update(ctx, current)
		}, 5*time.Second, 100*time.Millisecond).Should(Succeed())
		Eventually(func() bool {
			return apierrors.IsNotFound(k8sClient.Get(ctx, key, &v1alpha1.MeshSync{}))
		}, 5*time.Second, 100*time.Millisecond).Should(BeTrue())
	})
})

// TestMeshSyncReconcileTerminatingLegacyFinalizer is the MeshSync twin of
// TestBrokerReconcileTerminatingLegacyFinalizer: Reconcile is driven directly so
// the Terminating branch runs deterministically, and must neither keep the CR
// alive nor re-apply its Deployment.
func TestMeshSyncReconcileTerminatingLegacyFinalizer(t *testing.T) {
	tests := []struct {
		name       string
		finalizers []string
		wantGone   bool
	}{
		{name: "legacy finalizer only: deletion completes", finalizers: []string{legacyMeshsyncFinalizer}, wantGone: true},
		{name: "another writer's finalizer survives", finalizers: []string{legacyMeshsyncFinalizer, testHoldFinalizer}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			scheme := setupMeshSyncTestScheme(t)
			now := metav1.Now()
			meshsync := &v1alpha1.MeshSync{ObjectMeta: metav1.ObjectMeta{
				Name: testMeshSyncName, Namespace: testMesheryNamespace, UID: "meshsync-uid",
				DeletionTimestamp: &now, Finalizers: test.finalizers,
			}}
			c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&v1alpha1.MeshSync{}).WithObjects(meshsync).Build()
			r := &MeshSyncReconciler{Client: c, Scheme: scheme, Log: logr.Discard()}

			result, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(meshsync)})
			if err != nil || result.RequeueAfter != 0 {
				t.Fatalf("Reconcile() = %+v, %v; want a clean exit", result, err)
			}
			current := &v1alpha1.MeshSync{}
			err = c.Get(ctx, client.ObjectKeyFromObject(meshsync), current)
			if test.wantGone {
				if !apierrors.IsNotFound(err) {
					t.Fatalf("MeshSync must be removed once its legacy finalizer is stripped, got err=%v finalizers=%v", err, current.Finalizers)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if want := []string{testHoldFinalizer}; !slices.Equal(current.Finalizers, want) {
					t.Fatalf("finalizers = %v, want %v", current.Finalizers, want)
				}
			}
			deployments := &appsv1.DeploymentList{}
			if err := c.List(ctx, deployments, client.InNamespace(testMesheryNamespace)); err != nil {
				t.Fatal(err)
			}
			if len(deployments.Items) != 0 {
				t.Fatalf("a Terminating MeshSync must not have its Deployment re-applied, found %d", len(deployments.Items))
			}
		})
	}
}
