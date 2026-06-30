package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestReconcileTotal(t *testing.T) {
	ReconcileTotal.WithLabelValues("broker").Inc()
	count := testutil.CollectAndCount(ReconcileTotal)
	if count != 1 {
		t.Errorf("Expected 1 metric, got %d", count)
	}
}

func TestReconcileErrors(t *testing.T) {
	ReconcileErrors.WithLabelValues("broker").Inc()
	count := testutil.CollectAndCount(ReconcileErrors)
	if count != 1 {
		t.Errorf("Expected 1 metric, got %d", count)
	}
}
