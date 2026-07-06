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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Initialize test suite entrypoint
func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var (
	k8sClient client.Client
	testEnv   *envtest.Environment
	mgr       ctrl.Manager
	clientSet *kubernetes.Clientset
	// mgrCancel stops the manager; mgrDone closes once mgr.Start has returned.
	// AfterSuite cancels then waits on mgrDone so the manager releases its
	// apiserver watches before testEnv.Stop() tears down the control plane.
	mgrCancel context.CancelFunc
	mgrDone   chan struct{}
)

var _ = BeforeSuite(func(ctx SpecContext) {
	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))

	By("bootstrapping test environment")
	timeout := 2 * time.Minute
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
		},
		ControlPlaneStartTimeout: timeout,
		ControlPlaneStopTimeout:  timeout,
		AttachControlPlaneOutput: false,
		// Resolve the control-plane binaries from KUBEBUILDER_ASSETS (set by
		// `make test` via setup-envtest). Avoids a hard-coded, arch-specific
		// path that breaks on arm64/macOS. See Makefile `test` target.
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

	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 8443,
			Host: "", // listen on all interfaces
		}),
		LeaderElection: false,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(mgr).ToNot(BeNil())

	clientSet, err = kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(clientSet).ToNot(BeNil())

	brokerReconciler := &BrokerReconciler{
		Client:     mgr.GetClient(),
		KubeConfig: cfg,
		Clientset:  clientSet,
		Log:        ctrl.Log.WithName("controllers").WithName("Broker"),
		Scheme:     mgr.GetScheme(),
	}

	err = brokerReconciler.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	meshSyncReconciler := &MeshSyncReconciler{
		Client:     mgr.GetClient(),
		KubeConfig: cfg,
		Clientset:  clientSet,
		Log:        ctrl.Log.WithName("controllers").WithName("MeshSync"),
		Scheme:     mgr.GetScheme(),
	}

	err = meshSyncReconciler.SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	// +kubebuilder:scaffold:builder
	// Drive the manager with a cancellable context - not ctrl.SetupSignalHandler,
	// which only stops on an OS signal - so AfterSuite can stop it deterministically
	// before testEnv.Stop(). mgr.Start returns nil once the context is cancelled.
	var mgrCtx context.Context
	mgrCtx, mgrCancel = context.WithCancel(context.Background())
	mgrDone = make(chan struct{})
	go func() {
		defer GinkgoRecover()
		defer close(mgrDone)
		ctrl.Log.Info("starting manager")
		Expect(mgr.Start(mgrCtx)).To(Succeed(), "manager exited with an error")
	}()

	k8sClient, err = client.New(cfg, client.Options{Scheme: mgr.GetScheme()})

	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	crd := &apiv1.CustomResourceDefinition{}

	err = k8sClient.Get(ctx, types.NamespacedName{Name: "meshsyncs.meshery.io"}, crd)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Get(ctx, types.NamespacedName{Name: "brokers.meshery.io"}, crd)
	Expect(err).NotTo(HaveOccurred())
	Expect(crd.Spec.Names.Kind).To(Equal("Broker"))
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	// Stop the manager and wait for it to fully drain before stopping the control
	// plane. Draining releases the apiserver watches while the apiserver is still
	// up; stopping the control plane out from under a live manager instead leaves
	// testEnv.Stop() blocking on in-flight connections until it times out.
	if mgrCancel != nil {
		mgrCancel()
	}
	if mgrDone != nil {
		Eventually(mgrDone, 30*time.Second, 100*time.Millisecond).Should(BeClosed(),
			"manager did not shut down within the grace period")
	}
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
