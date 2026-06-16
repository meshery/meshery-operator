package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "meshery_operator_reconcile_total",
			Help: "Total number of reconciliations",
		},
		[]string{"controller"},
	)
	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "meshery_operator_reconcile_errors_total",
			Help: "Total number of reconciliation errors",
		},
		[]string{"controller"},
	)
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "meshery_operator_reconcile_duration_seconds",
			Help:    "Reconciliation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller"},
	)
)

func init() {
	metrics.Registry.MustRegister(ReconcileTotal, ReconcileErrors, ReconcileDuration)
}
