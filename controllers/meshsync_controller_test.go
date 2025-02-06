/*
Copyright 2020 Layer5, Inc.

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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
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
			namespace = "default"
			meshSync := &v1alpha1.MeshSync{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, meshSync)
			Expect(err).To(HaveOccurred())
		})
	})

	It("Creating a meshSync resource", func() {
		namespace = "default"
		meshSync := &v1alpha1.MeshSync{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "meshery.io/v1alpha1",
				Kind:       "MeshSync",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: namespace,
			},
			Spec: v1alpha1.MeshSyncSpec{
				Size: 1,
			}}

		Expect(k8sClient.Create(ctx, meshSync)).Should(Succeed())
	})

	It("Getting meshSync resource should be successful", func() {
		namespace = "default"
		meshSync := &v1alpha1.MeshSync{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, meshSync)
		Expect(err).ToNot(HaveOccurred())
		Expect(meshSync.Spec.Size).To(Equal(int32(1)))
	})

	It("Updating meshSync resource should be successful", func() {
		namespace = "default"
		meshSync := &v1alpha1.MeshSync{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, meshSync)
		Expect(err).ToNot(HaveOccurred())
		meshSync.Spec.Size = 2
		Expect(k8sClient.Update(ctx, meshSync)).Should(Succeed())

		By("Checking if the meshSync resource is updated")
		meshSync = &v1alpha1.MeshSync{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, meshSync)
		Expect(err).ToNot(HaveOccurred())
		Expect(meshSync.Spec.Size).To(Equal(int32(2)))
	})

	Context("Testing MeshSync's Cleanup logic", func() {
		It("Deleting meshSync resource should be succeeding", func() {
			namespace = "default"
			meshSync := &v1alpha1.MeshSync{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, meshSync)
			Expect(err).ToNot(HaveOccurred())
			Expect(k8sClient.Delete(ctx, meshSync)).Should(Succeed())
		})
	})

})
