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

// Package metrics defines and registers the Prometheus metrics exported by the
// Meshery Operator. Metrics are registered with the controller-runtime metrics
// registry so they are exposed automatically on the manager's metrics endpoint
// (default :8080/metrics) without any additional wiring.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// controllerLabel partitions the reconciliation metrics by controller (e.g.
// "broker", "meshsync").
const controllerLabel = "controller"

var (
	// ReconcileTotal counts reconciliations per controller.
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "meshery_operator_reconcile_total",
			Help: "Total number of reconciliations per controller.",
		},
		[]string{controllerLabel},
	)

	// ReconcileErrors counts reconciliations that returned an error, per controller.
	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "meshery_operator_reconcile_errors_total",
			Help: "Total number of reconciliation errors per controller.",
		},
		[]string{controllerLabel},
	)

	// ReconcileDuration tracks reconciliation latency in seconds, per controller.
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "meshery_operator_reconcile_duration_seconds",
			Help:    "Reconciliation duration in seconds per controller.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{controllerLabel},
	)
)

func init() {
	ctrlmetrics.Registry.MustRegister(ReconcileTotal, ReconcileErrors, ReconcileDuration)
}
