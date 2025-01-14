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

	"github.com/layer5io/meshery-operator/api/v1alpha1"
	brokerpackage "github.com/layer5io/meshery-operator/pkg/broker"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
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
			namespace = "default"
			broker := &v1alpha1.Broker{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, broker)
			Expect(err).To(HaveOccurred())
		})
	})

	It("Creating a broker resource", func() {
		namespace = "default"
		broker := &v1alpha1.Broker{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "meshery.layer5.io/v1alpha1",
				Kind:       "Broker",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: namespace,
			},
			Spec: v1alpha1.BrokerSpec{
				Size: 1,
			}}

		Expect(k8sClient.Create(ctx, broker)).Should(Succeed())
	})

	It("Getting broker resource should be successful", func() {
		namespace = "default"
		broker := &v1alpha1.Broker{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, broker)
		Expect(err).ToNot(HaveOccurred())
		Expect(broker.Spec.Size).To(Equal(int32(1)))
	})

	It("Updating broker resource should be successful", func() {
		namespace = "default"
		broker := &v1alpha1.Broker{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, broker)
		Expect(err).ToNot(HaveOccurred())
		broker.Spec.Size = 2
		Expect(k8sClient.Update(ctx, broker)).Should(Succeed())

		By("Checking if the broker resource is updated")
		broker = &v1alpha1.Broker{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, broker)
		Expect(err).ToNot(HaveOccurred())
		Expect(broker.Spec.Size).To(Equal(int32(2)))
	})

	Context("Testing Broker's brokerpackage.CheckHealth function", func() {
		It("Checking broker health functions", func() {
			namespace = "default"
			broker := &v1alpha1.Broker{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, broker)
			Expect(err).ToNot(HaveOccurred())
			By("Checking if the broker is healthy, it should return an error")
			Expect(brokerpackage.CheckHealth(ctx, broker, clientSet)).To(HaveOccurred())
			statefulSet := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "default",
				},
				Spec: v1.StatefulSetSpec{
					Replicas: &broker.Spec.Size,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "broker",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: "default",
							Labels: map[string]string{
								"app": "broker",
							},
						},
					},
				},
			}

			By("Creating statefulset resources for testing broker")
			err = k8sClient.Create(ctx, statefulSet)
			Expect(err).ToNot(HaveOccurred())
			By("Checking if the broker is healthy, it should be successful")
			Expect(brokerpackage.CheckHealth(ctx, broker, clientSet)).To(Succeed())
		})

	})

	Context("Testing Broker's Cleanup logic", func() {
		It("Deleting broker resource should be succeeding", func() {
			namespace = "default"
			broker := &v1alpha1.Broker{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, broker)
			Expect(err).ToNot(HaveOccurred())
			Expect(k8sClient.Delete(ctx, broker)).Should(Succeed())
		})
	})
})
