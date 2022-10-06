package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

	context := context.Background()

	const (
		URL          string = "https://layer5.io"
		str          string = "healthy"
		Reason       string = "Testcase"
		Message      string = "Message for testcase"
		PublishingTo string = "Publish for testcase"
		FileManager  string = "testcase-meshsync"

		Kind       string = "MeshSync"
		APIVersion string = "meshery.layer5.io/v1alpha1"
	)

	meshSync := &MeshSync{
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
		},
	}

	typeNamespace := types.NamespacedName{
		Namespace: "default",
		Name:      "default",
	}

	Context("The CURD case for the meshsync CRDs", func() {

		It("The meshsync CRDs create acticity should be succeed", func() {
			err := k8sClient.Create(context, meshSync)
			Expect(err).NotTo(HaveOccurred())
		})

		It("The meshsync CRDs get should be succeed", func() {

			mesheSyncGet := &MeshSync{}
			err := k8sClient.Get(context, typeNamespace, mesheSyncGet)
			Expect(err).NotTo(HaveOccurred())

			By("Checking the feild of the object we get")
			url := mesheSyncGet.Spec.Broker.Custom.URL
			Expect(url == URL).Should(BeTrue())

		})

		It("The meshsync CRDs update the spec of the resources", func() {
			meshSync.Spec.Size = 5
			err := k8sClient.Update(context, meshSync, &client.UpdateOptions{FieldManager: "testcase-meshsync"})
			Expect(err).NotTo(HaveOccurred())

			By("Get the size which updated")
			meshSyncGet := &MeshSync{}
			err = k8sClient.Get(context, typeNamespace, meshSyncGet)
			Expect(err).NotTo(HaveOccurred())

			By("Confirm the size equal to 5")
			Expect(meshSyncGet.Spec.Size == 5).Should(BeTrue())
		})

		It("The meshsync CRDs update the status of the resources", func() {
			meshSync.Status = MeshSyncStatus{
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

			err := k8sClient.Status().Update(context, meshSync, &client.UpdateOptions{FieldManager: FileManager})

			Expect(err).NotTo(HaveOccurred())
		})

		It("The meshsync CRDs remove the resources ", func() {

			By("Just delete the CRDs resources")
			err := k8sClient.Delete(context, meshSync)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
