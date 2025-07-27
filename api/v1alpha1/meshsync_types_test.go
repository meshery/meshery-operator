package v1alpha1

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/*
Here we do not want test CURD of kubernetes service, we want to confirm that:
* our CRDs resouces group version could be support by specific kubernetes server
* the specific value of the CRDs field can be workwell without pruning by the api-server to etcd
*/
var _ = Describe("The test case for the meshsync CRDs", func() {

	ctx := context.Background()

	const (
		URL          string = "https://layer5.io"
		str          string = "healthy"
		Reason       string = "Testcase"
		Message      string = "Message for testcase"
		PublishingTo string = "Publish for testcase"
		FileManager  string = "testcase-meshsync"

		Kind       string = "MeshSync"
		APIVersion string = "meshery.io/v1alpha1"
	)

	var meshSync *MeshSync
	var typeNamespace types.NamespacedName

	BeforeEach(func() {
		meshSync = &MeshSync{
			TypeMeta: metav1.TypeMeta{
				APIVersion: APIVersion,
				Kind:       Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "default",
			},
			Spec: MeshSyncSpec{
				Size: 2,
				Broker: MeshsyncBroker{
					Custom: CustomMeshsyncBroker{
						URL: URL,
					},
					Native: NativeMeshsyncBroker{
						Namespace: "default",
						Name:      "default",
					},
				},
				WatchList: corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1apha1",
						Kind:       "ConfigMap",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "watch-list",
						Namespace: "default",
					},
					Data: map[string]string{
						"blacklist": "",
						"whitelist": "[{\"Resource\":\"namespaces.v1.\",\"Events\":[\"ADDED\",\"DELETE\"]},{\"Resource\":\"replicasets.v1.apps\",\"Events\":[\"ADDED\",\"DELETE\"]},{\"Resource\":\"pods.v1.\",\"Events\":[\"MODIFIED\"]}]",
					},
				},
			},
		}

		typeNamespace = types.NamespacedName{
			Namespace: "default",
			Name:      "default",
		}

		err := fakeClient.Create(ctx, meshSync)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		existing := &MeshSync{}
		err := fakeClient.Get(ctx, typeNamespace, existing)
		if err == nil {
			err = fakeClient.Delete(ctx, existing)
			if !apierrors.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		}
	})

	Context("The CURD case for the meshsync CRDs", func() {

		It("The meshsync CRDs create acticity should be succeed", func() {
			By("Confirm the meshsync CRDs was created")
			meshSyncGet := &MeshSync{}
			err := fakeClient.Get(ctx, typeNamespace, meshSyncGet)
			Expect(err).NotTo(HaveOccurred())
			Expect(meshSyncGet.Name).To(Equal("default"))
		})

		It("The meshsync CRDs get should be succeed", func() {
			By("Get the meshsync CRDs")
			mesheSyncGet := &MeshSync{}
			err := fakeClient.Get(ctx, typeNamespace, mesheSyncGet)
			Expect(err).NotTo(HaveOccurred())

			By("Confirm the URL equal to https://layer5.io")
			url := mesheSyncGet.Spec.Broker.Custom.URL
			Expect(url == URL).Should(BeTrue())

			By("Confirm the config matches the expected listener and pipeline configs")
			configMap := mesheSyncGet.Spec.WatchList
			Expect(configMap).ShouldNot(BeNil())
			expectedConfigMap := corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1apha1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "watch-list",
					Namespace: "default",
				},
				Data: map[string]string{
					"blacklist": "",
					"whitelist": "[{\"Resource\":\"namespaces.v1.\",\"Events\":[\"ADDED\",\"DELETE\"]},{\"Resource\":\"replicasets.v1.apps\",\"Events\":[\"ADDED\",\"DELETE\"]},{\"Resource\":\"pods.v1.\",\"Events\":[\"MODIFIED\"]}]",
				},
			}
			Expect(configMap).To(Equal(expectedConfigMap))
		})

		It("The meshsync CRDs update the spec of the resources", func() {
			By("Get the latest version of the resource")
			existing := &MeshSync{}
			err := fakeClient.Get(ctx, typeNamespace, existing)
			Expect(err).NotTo(HaveOccurred())

			By("Update the size of the meshsync CRDs")
			existing.Spec.Size = 5
			err = fakeClient.Update(ctx, existing, &client.UpdateOptions{FieldManager: "testcase-meshsync"})
			Expect(err).NotTo(HaveOccurred())

			By("Get the latest version of meshsync CRDs")
			meshSyncGet := &MeshSync{}
			err = fakeClient.Get(ctx, typeNamespace, meshSyncGet)
			Expect(err).NotTo(HaveOccurred())

			By("Confirm the size equal to 5")
			Expect(meshSyncGet.Spec.Size == 5).Should(BeTrue())
		})

		It("The meshsync CRDs update the status of the resources", func() {
			By("Get the latest version of the resource")
			existing := &MeshSync{}
			err := fakeClient.Get(ctx, typeNamespace, existing)
			Expect(err).NotTo(HaveOccurred())

			fmt.Printf("Resource found: %s/%s, UID: %s, ResourceVersion: %s\n",
				existing.Namespace, existing.Name, existing.UID, existing.ResourceVersion)

			existing.Status = MeshSyncStatus{
				PublishingTo: PublishingTo,
				Conditions: []Condition{
					{
						Type:               ConditionType(str),
						Status:             ConditionStatus(str),
						ObservedGeneration: 1,
						Reason:             Reason,
						Message:            Message,
						LastTransitionTime: metav1.Now(),
						LastProbeTime:      metav1.Now(),
					},
				},
			}

			By("Update the status of the meshsync CRDs")
			err = fakeClient.Status().Update(ctx, existing)

			if err != nil {
				fmt.Printf("Status update failed: %v\n", err)
				// Try regular Get to see if resource still exists
				check := &MeshSync{}
				getErr := fakeClient.Get(ctx, typeNamespace, check)
				fmt.Printf("Resource still exists? %v\n", getErr == nil)
			}

			Expect(err).NotTo(HaveOccurred())
			Expect(existing.Status.PublishingTo == PublishingTo).Should(BeTrue())
		})

		It("The meshsyncList CRDs should be support by the kubernetes server", func() {
			By("Confirm the meshsync CRDs support by the kubernetes server")
			meshSyncList := &MeshSyncList{}
			err := fakeClient.List(ctx, meshSyncList, &client.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(meshSyncList.Items) > 0).Should(BeTrue())
		})

	})

	Context("The test coverage for delete the meshsync CRDs", func() {
		It("The meshsync CRDs remove the resources", func() {
			By("Delete the meshsync CRDs")
			err := fakeClient.Delete(ctx, meshSync)
			Expect(err).NotTo(HaveOccurred())

			By("Confirm deletion")
			deleted := &MeshSync{}
			err = fakeClient.Get(ctx, typeNamespace, deleted)
			Expect(apierrors.IsNotFound(err)).Should(BeTrue())
		})
	})
})
