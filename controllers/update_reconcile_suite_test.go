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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
				Kind:       "Broker",
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

		statefulSetKey := types.NamespacedName{Name: name, Namespace: namespace}
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
				Kind:       "MeshSync",
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
})
