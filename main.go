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
package main

import (
	"crypto/tls" // <-- Added
	"flag"
	"fmt"
	"os"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	"github.com/layer5io/meshery-operator/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	// "k8s.io/client-go/rest" // We get config from ctrl.GetConfigOrDie()
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	// --- Added metrics and filters imports ---
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// --- End added imports ---
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(mesheryv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr, namespace string
	var enableLeaderElection bool
	// --- Added flags for metrics security ---
	var secureMetrics bool
	var enableHTTP2 bool
	// --- End added flags ---

	// --- Updated metrics-addr flag name and default, added new flags ---
	// Defaulting to secure metrics on port 8443, as is standard now.
	flag.StringVar(
		&metricsAddr,
		"metrics-bind-address",
		":8443",
		"The address the metric endpoint binds to.",
	)
	// Secure metrics enabled by default.
	flag.BoolVar(
		&secureMetrics,
		"secure-metrics",
		true,
		"Enable secure serving for metrics.",
	)
	// HTTP/2 disabled by default for security.
	flag.BoolVar(
		&enableHTTP2,
		"enable-http2",
		false,
		"Enable HTTP/2 for the metrics and webhook servers. Recommended false.",
	)
	// --- End flag updates ---

	flag.StringVar(
		&namespace,
		"namespace",
		"meshery",
		"The namespace operator is deployed to.",
	)
	flag.BoolVar(
		&enableLeaderElection,
		"enable-leader-election",
		false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)
	flag.Parse()

	// Use Development=true for more verbose logs during development/debugging
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true})))

	// --- Configure TLS options ---
	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		disableHTTP2 := func(c *tls.Config) {
			setupLog.Info("disabling http/2")
			c.NextProtos = []string{"http/1.1"}
		}
		tlsOpts = append(tlsOpts, disableHTTP2)
	}
	// --- End TLS options ---

	// --- Configure Metrics Server Options ---
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// Add the auth filter *only* if secure serving is enabled
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
		setupLog.Info(
			"Metrics endpoint protection enabled using controller-runtime filters",
			"address",
			metricsAddr,
		)
	} else {
		// Log if metrics are insecure (should be rare if default is true)
		setupLog.Info(
			"Metrics endpoint is serving insecurely",
			"address",
			metricsAddr,
		)
	}
	// --- End Metrics Server Options ---

	// --- Configure Webhook Server (apply TLS options) ---
	webhookServer := webhook.NewServer(webhook.Options{
		Port:    9443, // Default webhook port
		TLSOpts: tlsOpts,
	})
	// --- End Webhook Server ---

	opID := uuid.NewUUID()
	// Get config before creating manager
	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		// --- Pass the configured metrics options ---
		Metrics: metricsServerOptions,
		// --- Pass the configured webhook server ---
		WebhookServer:           webhookServer,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        fmt.Sprintf("operator-%s.meshery.io", opID),
		LeaderElectionNamespace: namespace,
		// HealthProbeBindAddress: probeAddr, // Add if you use health probes
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// --- Get Kubernetes clientset (no change needed) ---
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "unable to initialize clientset")
		os.Exit(1)
	}

	// --- Setup Reconcilers (no change needed) ---
	mReconciler := &controllers.MeshSyncReconciler{
		KubeConfig: cfg, // Use cfg directly
		Client:     mgr.GetClient(),
		Clientset:  clientset,
		Log:        ctrl.Log.WithName("MeshSync"),
		Scheme:     mgr.GetScheme(),
	}

	bReconciler := &controllers.BrokerReconciler{
		KubeConfig: cfg, // Use cfg directly
		Client:     mgr.GetClient(),
		Clientset:  clientset,
		Log:        ctrl.Log.WithName("Broker"),
		Scheme:     mgr.GetScheme(),
	}

	if err = mReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "MeshSync")
		os.Exit(1)
	}
	if err = bReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "Broker")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
