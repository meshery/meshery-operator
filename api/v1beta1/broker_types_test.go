/*
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
package v1beta1

// Import go packages

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("The test cases for customize resource: Broker", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		BrokerName      string = "broker-test"
		BrokerNamespace string = "broker-test-namespace"
	)

	// create a Broker object
	broker := &Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BrokerName,
			Namespace: BrokerNamespace,
		},
		Spec: BrokerSpec{
			Size: 1,
		},
	}

	// use fake client to test the Broker object
	Context("When creating a Broker object", func() {

		ctx := context.Background()
		It("should create a Broker object successfully", func() {
			Expect(fakeClient.Create(ctx, broker)).Should(Succeed())
		})

		It("should get the Broker object successfully", func() {
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Name:      BrokerName,
				Namespace: BrokerNamespace,
			}, broker)).Should(Succeed())
		})

		It("should update the Broker object successfully", func() {
			broker.Spec.Size = 2
			Expect(fakeClient.Update(ctx, broker)).Should(Succeed())
		})

		// check broker.Spec.Size is 2
		It("should get the Broker object successfully", func() {
			broker2 := &Broker{}
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Name:      BrokerName,
				Namespace: BrokerNamespace,
			}, broker2)).Should(Succeed())
			Expect(broker2.Spec.Size).Should(Equal(int32(2)))
		})

		// check broker list
		It("should list the Broker object successfully", func() {
			brokerList := &BrokerList{}
			Expect(fakeClient.List(ctx, brokerList)).Should(Succeed())
			Expect(len(brokerList.Items)).Should(Equal(1))
		})

		It("should delete the Broker object successfully", func() {
			Expect(fakeClient.Delete(ctx, broker)).Should(Succeed())
		})
	})
})
