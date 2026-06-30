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

package broker

import (
	"context"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultNamespace = "default"

var _ = Describe("Broker funtions test cases", func() {

	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.TODO()
	})

	Context("Test for GetObjects function", func() {
		It("should return the ordered slice of objects", func() {
			m := &mesheryv1alpha1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: mesheryv1alpha1.BrokerSpec{
					Size: 1,
				},
			}
			objs := GetObjects(m)
			Expect(objs).To(HaveLen(4))

			By("ConfigMaps and Service must precede the StatefulSet")
			_, lastIsStatefulSet := objs[len(objs)-1].(*v1.StatefulSet)
			Expect(lastIsStatefulSet).To(BeTrue())

			By("the StatefulSet must carry the Broker name and namespace")
			var sts *v1.StatefulSet
			for _, o := range objs {
				Expect(o).ToNot(BeNil())
				if s, ok := o.(*v1.StatefulSet); ok {
					sts = s
				}
			}
			Expect(sts).ToNot(BeNil())
			Expect(sts.GetName()).To(Equal(m.Name))
			Expect(sts.GetNamespace()).To(Equal(m.Namespace))
		})
	})

	Context("Test for CheckHealth function", func() {
		It("should be unhealthy until ReadyReplicas reaches the desired count", func() {
			namespace := defaultNamespace
			name := defaultNamespace
			m := &mesheryv1alpha1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: mesheryv1alpha1.BrokerSpec{
					Size: 1,
				},
			}

			s := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
				Spec: v1.StatefulSetSpec{
					Replicas: &m.Spec.Size,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							appLabelKey: brokerComponent,
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
							Labels: map[string]string{
								appLabelKey: brokerComponent,
							},
						},
						Spec: corev1.PodSpec{
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
			By("Creating the statefulset (no ready replicas yet)")
			Expect(k8sClient.Create(ctx, s)).To(Succeed())

			By("Health must fail while ReadyReplicas is 0")
			Expect(CheckHealth(ctx, m, k8sClient)).ToNot(Succeed())

			By("Driving the StatefulSet status to ready via the status subresource")
			s.Status.Replicas = 1
			s.Status.ReadyReplicas = 1
			Expect(k8sClient.Status().Update(ctx, s)).To(Succeed())

			By("Health must now succeed")
			Expect(CheckHealth(ctx, m, k8sClient)).To(Succeed())
		})
	})

	Context("Test for GetEndpoint function", func() {
		It("should derive the endpoint from a NodePort Service without network I/O", func() {
			name := defaultNamespace
			namespace := defaultNamespace
			m := &mesheryv1alpha1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: mesheryv1alpha1.BrokerSpec{
					Size: 1,
				},
			}

			By("Create the broker Service first")
			s := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{Name: clientPortName, Port: 4222, NodePort: 30002},
						{Name: "monitor", Port: 8222},
					},
				},
			}
			Expect(k8sClient.Create(ctx, s)).To(Succeed())

			apiServerURL := "http://localhost:8080"
			Expect(GetEndpoint(ctx, m, k8sClient, apiServerURL)).To(Succeed())

			By("checking m.status.endpoint")
			Expect(m.Status.Endpoint.External).To(Equal("localhost:30002"))
			Expect(m.Status.Endpoint.Internal).To(ContainSubstring(":4222"))
		})
	})
})
