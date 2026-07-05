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

const (
	defaultNamespace = "default"
	testBrokerName   = "meshery-broker"
)

var _ = Describe("Broker funtions test cases", func() {

	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.TODO()
	})

	Context("Test for CheckHealth function", func() {
		It("should be unhealthy until ReadyReplicas reaches the desired count", func() {
			m := &mesheryv1alpha1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testBrokerName,
					Namespace: defaultNamespace,
				},
				Spec: mesheryv1alpha1.BrokerSpec{
					Size: 1,
				},
			}

			// The operator reads the NATS StatefulSet by its fixed chart name.
			s := &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: defaultNamespace,
					Name:      natsServiceName,
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
							Name: natsServiceName,
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
			m := &mesheryv1alpha1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testBrokerName,
					Namespace: defaultNamespace,
				},
				Spec: mesheryv1alpha1.BrokerSpec{
					Size: 1,
				},
			}

			By("Create the broker client Service under its fixed chart name")
			s := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      natsServiceName,
					Namespace: defaultNamespace,
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{Name: "nats", Port: 4222, NodePort: 30002},
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

	Context("Test for GenerateToken function", func() {
		It("should produce a token that is safe to embed unquoted in nats.conf", func() {
			// Regression: the token is injected unquoted (`token: $NATS_TOKEN`) and
			// NATS re-lexes it. A token that starts with a digit can be misparsed as
			// a number (e.g. "758e126b..." -> scientific notation), crashing NATS.
			// The token must therefore always start with a letter so NATS lexes it
			// as a string. Run many iterations since the failure was token-dependent.
			for i := 0; i < 500; i++ {
				token, err := GenerateToken()
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(BeEmpty())
				first := token[0]
				Expect(first >= '0' && first <= '9').To(
					BeFalse(),
					"token %q must not start with a digit (NATS may misparse it as a number)", token,
				)
			}
		})
	})
})
