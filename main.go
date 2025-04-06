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
	"crypto/tls"
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	"github.com/layer5io/meshery-operator/controllers"

	"k8s.io/client-go/kubernetes"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// +kubebuilder:scaffold:scheme
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(mesheryv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr, namespace string
	var enableLeaderElection bool
	var secureMetrics bool

	flag.StringVar(
		&metricsAddr,
		"metrics-bind-address",
		":8443",
		"The address the metric endpoint binds to.",
	)

	flag.BoolVar(
		&secureMetrics,
		"secure-metrics",
		true,
		"Enable secure serving for metrics.",
	)

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
			"Enabling this will ensure there is only one active controller manager.")

	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{})))

	tlsOpts := []func(*tls.Config){}

	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	webhookServer := webhook.NewServer(webhook.Options{
		Port:    9443, // Default webhook port
		TLSOpts: tlsOpts,
	})

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

	opID := uuid.NewUUID()

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		Metrics:                 metricsServerOptions,
		WebhookServer:           webhookServer,
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        fmt.Sprintf("operator-%s.meshery.io", opID),
		LeaderElectionNamespace: namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "unable to initialize clientset")
		os.Exit(1)
	}

	mReconciler := &controllers.MeshSyncReconciler{
		KubeConfig: cfg,
		Client:     mgr.GetClient(),
		Clientset:  clientset,
		Log:        ctrl.Log.WithName("MeshSync"),
		Scheme:     mgr.GetScheme(),
	}

	bReconciler := &controllers.BrokerReconciler{
		KubeConfig: cfg,
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
