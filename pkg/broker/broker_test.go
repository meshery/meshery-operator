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

var _ = Describe("Broker funtions test cases", func() {

	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.TODO()
	})

	Context("Test for GetObjects function", func() {
		It("should return the map of objects", func() {
			m := &mesheryv1alpha1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: mesheryv1alpha1.BrokerSpec{
					Size: 1,
				},
			}
			obj := GetObjects(m)
			Expect(obj).ToNot(BeNil())

			By("checking server config")
			Expect(obj[ServerConfig]).ToNot(BeNil())

			By("checking account config")
			Expect(obj[AccountConfig]).ToNot(BeNil())

			By("checking server object, namespace and name, replicas")
			Expect(obj[ServerObject]).ToNot(BeNil())
			Expect(obj[ServerObject].GetNamespace()).To(Equal(m.Namespace))
			Expect(obj[ServerObject].GetName()).To(Equal(m.Name))
		})
	})

	Context("Test for CheckHealth function", func() {
		It("should return nil", func() {

			namespace := "default"
			name := "default"
			m := &mesheryv1alpha1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: mesheryv1alpha1.BrokerSpec{
					Size: 1,
				},
			}

			// create a statefulSet object in the cluster
			s := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
				Spec: v1.StatefulSetSpec{
					Replicas: &m.Spec.Size,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "broker",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
							Labels: map[string]string{
								"app": "broker",
							},
						},
					},
				},
			}
			By("Creating statefulset resources for testing broker")
			err := k8sClient.Create(ctx, s)
			Expect(err).ToNot(HaveOccurred())
			By("Checking if the broker is healthy, it should be successful")
			Expect(CheckHealth(ctx, m, k8sClient)).To(Succeed())

		})
	})

	Context("Test for GetEndpoint function", func() {
		It("should return the endpoint", func() {

			name := "default"
			namespace := "default"
			m := &mesheryv1alpha1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: mesheryv1alpha1.BrokerSpec{
					Size: 1,
				},
			}

			By("Create service first")
			s := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Name: "http",
							Port: 8080,
						},
						{
							NodePort: 30002,
							Name:     "grpc",
							Port:     8082,
						},
					},
				},
			}
			err := k8sClient.Create(ctx, s)
			Expect(err).ToNot(HaveOccurred())

			url := "http://localhost:8080"
			Expect(GetEndpoint(ctx, m, k8sClient, url)).ShouldNot(HaveOccurred())

			By("checking m.status.endpoint")
			Expect(m.Status.Endpoint.External).To(Equal("localhost:30002"))
			Expect(m.Status.Endpoint.Internal).Should(ContainSubstring("8082"))
		})
	})
})
