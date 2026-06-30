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

package main

import (
	"flag"
	"os"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	"github.com/meshery/meshery-operator/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// RBAC for the controller-runtime metrics authn/authz filter: it delegates
// bearer-token authentication and SubjectAccessReview authorization to the API
// server, so the operator's ServiceAccount must be allowed to create both.
// +kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// leaderElectionID is the name of the Lease used to coordinate leader
// election. It MUST be stable across all replicas and restarts; otherwise
// every manager would acquire its own uniquely-named lease and never contend,
// defeating leader election entirely.
const leaderElectionID = "meshery-operator-leader.meshery.io"

func init() {
	// +kubebuilder:scaffold:scheme
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(mesheryv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr, probeAddr, namespace string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8443", "The address the metric endpoint binds to (served over TLS with authn/authz).")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the health probe endpoint binds to.")
	flag.StringVar(&namespace, "namespace", "meshery", "The namespace operator is deployed to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{})))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
			// Serve metrics over TLS (controller-runtime self-signs when no
			// CertDir is provided) and gate access behind the API server's
			// authn + SubjectAccessReview authz, replacing the retired
			// kube-rbac-proxy sidecar (WS-5).
			SecureServing:  true,
			FilterProvider: filters.WithAuthenticationAndAuthorization,
		},
		HealthProbeBindAddress: probeAddr,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        leaderElectionID,
		LeaderElectionNamespace: namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to initialize clientset")
		os.Exit(1)
	}

	mReconciler := &controllers.MeshSyncReconciler{
		KubeConfig: mgr.GetConfig(),
		Client:     mgr.GetClient(),
		Clientset:  clientset,
		Log:        ctrl.Log.WithName("MeshSync"),
		Scheme:     mgr.GetScheme(),
	}

	bReconciler := &controllers.BrokerReconciler{
		KubeConfig: mgr.GetConfig(),
		Client:     mgr.GetClient(),
		Clientset:  clientset,
		Log:        ctrl.Log.WithName("Broker"),
		Scheme:     mgr.GetScheme(),
	}

	if err = mReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MeshSync")
		os.Exit(1)
	}
	if err = bReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Broker")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
