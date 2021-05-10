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
	"flag"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	mesheryv1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	"github.com/layer5io/meshery-operator/controllers"
	"github.com/layer5io/meshkit/logger"
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
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	// Initialize Logger instance
	log, err := logger.New("meshery-operator", logger.Options{
		Format: logger.SyslogLogFormat,
	})
	if err != nil {
		setupLog.Error(err, "unable to initialize logger")
		os.Exit(1)
	}

	ctrl.SetLogger(log.ControllerLogger())

	opId := uuid.NewUUID()
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   fmt.Sprintf("operator-%s.meshery.layer5.io", opId),
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

	if err = (&controllers.MeshSyncReconciler{
		KubeConfig: mgr.GetConfig(),
		Client:     mgr.GetClient(),
		Clientset:  clientset,
		Log:        ctrl.Log,
		Scheme:     mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "MeshSync")
		os.Exit(1)
	}
	if err = (&controllers.BrokerReconciler{
		KubeConfig: mgr.GetConfig(),
		Client:     mgr.GetClient(),
		Clientset:  clientset,
		Log:        ctrl.Log,
		Scheme:     mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
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
