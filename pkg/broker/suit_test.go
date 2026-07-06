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
	"path/filepath"
	"strings"
	"testing"
	"time"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Initialize test suite entrypoint
func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Suite")
}

var (
	k8sClient client.Client
	testEnv   *envtest.Environment
	clientSet *kubernetes.Clientset
)

var _ = BeforeSuite(func(ctx SpecContext) {
	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))

	By("bootstrapping test environment")
	timeout := 3 * time.Minute
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ControlPlaneStartTimeout: timeout,
		ControlPlaneStopTimeout:  timeout,
		AttachControlPlaneOutput: false,
		// Binaries resolved from KUBEBUILDER_ASSETS (set by `make test`).
	}

	var cfg *rest.Config
	var err error
	done := make(chan interface{})
	go func() {
		defer GinkgoRecover()
		cfg, err = testEnv.Start()
		close(done)
	}()
	Eventually(done).WithContext(ctx).WithTimeout(timeout).Should(BeClosed())
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()

	Expect(mesheryv1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(k8sscheme.AddToScheme(scheme)).To(Succeed())
	Expect(apiv1.AddToScheme(scheme)).To(Succeed())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	clientSet, err = kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(clientSet).NotTo(BeNil())

	crd := &apiv1.CustomResourceDefinition{}

	err = k8sClient.Get(ctx, types.NamespacedName{Name: "meshsyncs.meshery.io"}, crd)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Get(ctx, types.NamespacedName{Name: "brokers.meshery.io"}, crd)
	Expect(err).NotTo(HaveOccurred())
	Expect(crd.Spec.Names.Kind).To(Equal("Broker"))
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	// A control-plane stop timeout is a cleanup-phase artifact - seen with the
	// darwin envtest binaries, where kube-apiserver does not exit on the stop
	// signal. The specs have already run and the orphaned processes are reaped
	// when this test binary exits, so it must not fail an otherwise-green suite;
	// log it and move on. Every other Stop error is surfaced normally.
	if err := testEnv.Stop(); err != nil {
		if strings.Contains(err.Error(), "timeout waiting for process") {
			GinkgoWriter.Printf("AfterSuite: control plane did not stop cleanly (ignored): %v\n", err)
			return
		}
		Expect(err).NotTo(HaveOccurred())
	}
})
