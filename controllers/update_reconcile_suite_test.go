package controllers

import (
	"context"
	"fmt"
	"time"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Controller update reconciliation", func() {
	ctx := context.Background()

	createNamespace := func(name string) {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
		err := k8sClient.Create(ctx, ns)
		if err != nil {
			Expect(err.Error()).To(ContainSubstring("already exists"))
		}
	}

	It("updates an existing broker statefulset when the broker spec changes", func() {
		namespace := fmt.Sprintf("broker-update-%d", GinkgoRandomSeed())
		name := "broker-update"
		createNamespace(namespace)

		broker := &mesheryv1alpha1.Broker{
			TypeMeta: metav1.TypeMeta{
				APIVersion: testAPIVersion,
				Kind:       testBrokerKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: mesheryv1alpha1.BrokerSpec{
				Size: 1,
			},
		}
		Expect(k8sClient.Create(ctx, broker)).To(Succeed())

		// Broker-owned objects use the fixed chart release name, not the CR name.
		statefulSetKey := types.NamespacedName{Name: natsObjectName, Namespace: namespace}
		Eventually(func(g Gomega) {
			statefulSet := &appsv1.StatefulSet{}
			g.Expect(k8sClient.Get(ctx, statefulSetKey, statefulSet)).To(Succeed())
			g.Expect(statefulSet.Spec.Replicas).ToNot(BeNil())
			g.Expect(*statefulSet.Spec.Replicas).To(Equal(int32(1)))
		}, 15*time.Second, 250*time.Millisecond).Should(Succeed())

		Eventually(func() error {
			current := &mesheryv1alpha1.Broker{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, current); err != nil {
				return err
			}
			current.Spec.Size = 2
			return k8sClient.Update(ctx, current)
		}, 5*time.Second, 250*time.Millisecond).Should(Succeed())

		Eventually(func(g Gomega) {
			statefulSet := &appsv1.StatefulSet{}
			g.Expect(k8sClient.Get(ctx, statefulSetKey, statefulSet)).To(Succeed())
			g.Expect(statefulSet.Spec.Replicas).ToNot(BeNil())
			g.Expect(*statefulSet.Spec.Replicas).To(Equal(int32(2)))
		}, 15*time.Second, 250*time.Millisecond).Should(Succeed())
	})

	It("updates an existing meshsync deployment when the meshsync spec changes", func() {
		namespace := fmt.Sprintf("meshsync-update-%d", GinkgoRandomSeed())
		name := "meshsync-update"
		createNamespace(namespace)

		meshsync := &mesheryv1alpha1.MeshSync{
			TypeMeta: metav1.TypeMeta{
				APIVersion: testAPIVersion,
				Kind:       testMeshSyncKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: mesheryv1alpha1.MeshSyncSpec{
				Size: 1,
				Broker: mesheryv1alpha1.MeshsyncBroker{
					Custom: mesheryv1alpha1.CustomMeshsyncBroker{
						URL: "nats://broker-old:4222",
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, meshsync)).To(Succeed())

		deploymentKey := types.NamespacedName{Name: name, Namespace: namespace}
		Eventually(func(g Gomega) {
			deployment := &appsv1.Deployment{}
			g.Expect(k8sClient.Get(ctx, deploymentKey, deployment)).To(Succeed())
			g.Expect(deployment.Spec.Replicas).ToNot(BeNil())
			g.Expect(*deployment.Spec.Replicas).To(Equal(int32(1)))
			g.Expect(deployment.Spec.Template.Spec.Containers).ToNot(BeEmpty())
			g.Expect(deployment.Spec.Template.Spec.Containers[0].Env).ToNot(BeEmpty())
			g.Expect(deployment.Spec.Template.Spec.Containers[0].Env[0].Value).To(Equal("nats://broker-old:4222"))
		}, 15*time.Second, 250*time.Millisecond).Should(Succeed())

		Eventually(func() error {
			current := &mesheryv1alpha1.MeshSync{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, current); err != nil {
				return err
			}
			current.Spec.Size = 2
			current.Spec.Broker.Custom.URL = "nats://broker-new:4222"
			return k8sClient.Update(ctx, current)
		}, 5*time.Second, 250*time.Millisecond).Should(Succeed())

		Eventually(func(g Gomega) {
			deployment := &appsv1.Deployment{}
			g.Expect(k8sClient.Get(ctx, deploymentKey, deployment)).To(Succeed())
			g.Expect(deployment.Spec.Replicas).ToNot(BeNil())
			g.Expect(*deployment.Spec.Replicas).To(Equal(int32(2)))
			g.Expect(deployment.Spec.Template.Spec.Containers).ToNot(BeEmpty())
			g.Expect(deployment.Spec.Template.Spec.Containers[0].Env).ToNot(BeEmpty())
			g.Expect(deployment.Spec.Template.Spec.Containers[0].Env[0].Value).To(Equal("nats://broker-new:4222"))
		}, 15*time.Second, 250*time.Millisecond).Should(Succeed())
	})

	It("reconciles the broker Service type in place when spec.service.type changes", func() {
		namespace := fmt.Sprintf("broker-net-%d", GinkgoRandomSeed())
		name := "broker-net"
		createNamespace(namespace)

		broker := &mesheryv1alpha1.Broker{
			TypeMeta:   metav1.TypeMeta{APIVersion: testAPIVersion, Kind: testBrokerKind},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: mesheryv1alpha1.BrokerSpec{
				Size:    1,
				Service: mesheryv1alpha1.BrokerServiceSpec{Type: corev1.ServiceTypeClusterIP},
			},
		}
		Expect(k8sClient.Create(ctx, broker)).To(Succeed())

		svcKey := types.NamespacedName{Name: natsObjectName, Namespace: namespace}
		By("the NATS Service is created as ClusterIP")
		Eventually(func(g Gomega) {
			svc := &corev1.Service{}
			g.Expect(k8sClient.Get(ctx, svcKey, svc)).To(Succeed())
			g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
		}, 15*time.Second, 250*time.Millisecond).Should(Succeed())

		By("patching the Broker to NodePort reconciles the live Service in place (no recreation)")
		Eventually(func() error {
			current := &mesheryv1alpha1.Broker{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, current); err != nil {
				return err
			}
			current.Spec.Service.Type = corev1.ServiceTypeNodePort
			return k8sClient.Update(ctx, current)
		}, 5*time.Second, 250*time.Millisecond).Should(Succeed())

		Eventually(func(g Gomega) {
			svc := &corev1.Service{}
			g.Expect(k8sClient.Get(ctx, svcKey, svc)).To(Succeed())
			g.Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
		}, 15*time.Second, 250*time.Millisecond).Should(Succeed())
	})

	It("rejects loadBalancerClass on a non-LoadBalancer service via CEL", func() {
		namespace := fmt.Sprintf("broker-cel-%d", GinkgoRandomSeed())
		name := "broker-cel"
		createNamespace(namespace)

		class := "internal"
		broker := &mesheryv1alpha1.Broker{
			TypeMeta:   metav1.TypeMeta{APIVersion: testAPIVersion, Kind: testBrokerKind},
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: mesheryv1alpha1.BrokerSpec{
				Size: 1,
				Service: mesheryv1alpha1.BrokerServiceSpec{
					Type:              corev1.ServiceTypeClusterIP,
					LoadBalancerClass: &class,
				},
			},
		}
		err := k8sClient.Create(ctx, broker)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("loadBalancerClass is only valid when service type is LoadBalancer"))
	})

	It("rejects a malformed loadBalancerSourceRanges entry at admission, in every served version", func() {
		namespace := fmt.Sprintf("broker-cidr-%d", GinkgoRandomSeed())
		createNamespace(namespace)

		newBroker := func(version string, ranges ...interface{}) *unstructured.Unstructured {
			broker := &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "meshery.io/" + version,
				"kind":       testBrokerKind,
				"metadata":   map[string]interface{}{"name": "broker-cidr-" + version, "namespace": namespace},
				"spec": map[string]interface{}{
					"size": int64(1),
					"service": map[string]interface{}{
						"type":                     string(corev1.ServiceTypeLoadBalancer),
						"loadBalancerSourceRanges": ranges,
					},
				},
			}}
			return broker
		}

		for _, version := range []string{"v1alpha1", "v1alpha2"} {
			By("a server-side dry run of an unrendered template placeholder fails validation (" + version + ")")
			err := k8sClient.Create(ctx, newBroker(version, "203.0.113.0/24", "${PLAYGROUND_EGRESS_CIDR}"), client.DryRunAll)
			Expect(apierrors.IsInvalid(err)).To(BeTrue(), "want an Invalid admission error, got %v", err)
			Expect(err.Error()).To(ContainSubstring("spec.service.loadBalancerSourceRanges[1]"))

			By("IPv4 and IPv6 CIDRs are admitted (" + version + ")")
			Expect(k8sClient.Create(ctx, newBroker(version, "203.0.113.0/24", "2001:db8::/32", "::ffff:192.0.2.1/128"), client.DryRunAll)).To(Succeed())
		}
	})
})

var _ = Describe("Broker upgrade cleanup", func() {
	It("cleans up legacy broker resources after publishing the replacement endpoint", func() {
		ctx := context.Background()
		namespace := fmt.Sprintf("broker-migration-%d", GinkgoRandomSeed())
		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})).To(Succeed())
		broker := &mesheryv1alpha1.Broker{
			ObjectMeta: metav1.ObjectMeta{Name: "custom-broker", Namespace: namespace},
			Spec:       mesheryv1alpha1.BrokerSpec{Size: 1},
		}
		Expect(k8sClient.Create(ctx, broker)).To(Succeed())
		owner := metav1.NewControllerRef(broker, mesheryv1alpha1.GroupVersion.WithKind("Broker"))
		labels := legacyBrokerTestLabels()
		one := int32(1)
		old := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: broker.Name, Namespace: namespace, Labels: labels, OwnerReferences: []metav1.OwnerReference{*owner}},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &one, ServiceName: broker.Name, Selector: &metav1.LabelSelector{MatchLabels: labels},
				Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "nats", Image: "nats:2.8.2-alpine3.15"}}}},
			},
		}
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: broker.Name, Namespace: namespace, Labels: labels, OwnerReferences: []metav1.OwnerReference{*owner}},
			Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer, Selector: labels, Ports: []corev1.ServicePort{{Name: testNatsClientPort, Port: 4222}}},
		}
		Expect(k8sClient.Create(ctx, old)).To(Succeed())
		Expect(k8sClient.Create(ctx, service)).To(Succeed())
		newKey := types.NamespacedName{Name: natsObjectName, Namespace: namespace}
		Eventually(func() error {
			return k8sClient.Get(ctx, newKey, &appsv1.StatefulSet{})
		}, 15*time.Second, 100*time.Millisecond).Should(Succeed())
		By("retaining old resources while the replacement is unready")
		Consistently(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: broker.Name, Namespace: namespace}, &corev1.Service{})
		}, time.Second, 100*time.Millisecond).Should(Succeed())
		By("driving the replacement rollout status because envtest has no kubelet")
		Eventually(func() error {
			current := &appsv1.StatefulSet{}
			if err := k8sClient.Get(ctx, newKey, current); err != nil {
				return err
			}
			current.Status.ObservedGeneration = current.Generation
			current.Status.Replicas, current.Status.ReadyReplicas, current.Status.UpdatedReplicas = one, one, one
			current.Status.CurrentRevision, current.Status.UpdateRevision = "replacement", "replacement"
			return k8sClient.Status().Update(ctx, current)
		}, 5*time.Second, 100*time.Millisecond).Should(Succeed())
		By("publishing the new endpoint and removing only obsolete owned children")
		Eventually(func(g Gomega) {
			current := &mesheryv1alpha1.Broker{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: broker.Name, Namespace: namespace}, current)).To(Succeed())
			g.Expect(current.Status.Endpoint.Internal).ToNot(BeEmpty())
			g.Expect(apierrors.IsNotFound(k8sClient.Get(ctx, types.NamespacedName{Name: broker.Name, Namespace: namespace}, &appsv1.StatefulSet{}))).To(BeTrue())
			g.Expect(apierrors.IsNotFound(k8sClient.Get(ctx, types.NamespacedName{Name: broker.Name, Namespace: namespace}, &corev1.Service{}))).To(BeTrue())
			g.Expect(k8sClient.Get(ctx, newKey, &appsv1.StatefulSet{})).To(Succeed())
			g.Expect(k8sClient.Get(ctx, newKey, &corev1.Service{})).To(Succeed())
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())
	})
	It("checks stored Secret defaults, environment ordering and duplicate definitions", func() {
		ctx := context.Background()
		namespace := fmt.Sprintf("broker-env-semantic-%d", GinkgoRandomSeed())
		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})).To(Succeed())
		f := newBrokerCleanupFixture()
		deployment := f.addMeshSync()
		want := deployment.Spec.Template.Spec.Containers[0].DeepCopy()
		deployment.Namespace = namespace
		deployment.OwnerReferences = nil
		deployment.Generation = 0
		optional := false
		brokerCleanupTestEnv(&deployment.Spec.Template.Spec.Containers[0], testNatsTokenEnv).ValueFrom.SecretKeyRef.Optional = &optional
		Expect(k8sClient.Create(ctx, deployment)).To(Succeed())
		stored := &appsv1.Deployment{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: namespace}, stored)).To(Succeed())
		selector := brokerCleanupTestEnv(&stored.Spec.Template.Spec.Containers[0], testNatsTokenEnv).ValueFrom.SecretKeyRef
		Expect(selector.Optional).ToNot(BeNil())
		Expect(*selector.Optional).To(BeFalse())
		Expect(brokerEnvMatches(*want, stored.Spec.Template.Spec.Containers)).To(BeTrue())

		By("rejecting an API-stored URL placed before its token reference")
		container := &stored.Spec.Template.Spec.Containers[0]
		tokenEnv := brokerCleanupTestEnv(container, testNatsTokenEnv)
		urlEnv := brokerCleanupTestEnv(container, testBrokerURLEnv)
		*tokenEnv, *urlEnv = *urlEnv, *tokenEnv
		Expect(k8sClient.Update(ctx, stored)).To(Succeed())
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: namespace}, stored)).To(Succeed())
		Expect(brokerEnvMatches(*want, stored.Spec.Template.Spec.Containers)).To(BeFalse())

		By("rejecting an API-stored duplicate URL that overrides the earlier value")
		stored.Spec.Template.Spec.Containers[0].Env = want.DeepCopy().Env
		stored.Spec.Template.Spec.Containers[0].Env = append(stored.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{Name: testBrokerURLEnv, Value: testOldBrokerURL})
		Expect(k8sClient.Update(ctx, stored)).To(Succeed())
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: namespace}, stored)).To(Succeed())
		Expect(brokerEnvMatches(*want, stored.Spec.Template.Spec.Containers)).To(BeFalse())
	})

})
