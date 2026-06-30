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

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// TestReconcileCountersIncrement verifies the counter vectors track values
// independently per controller label.
func TestReconcileCountersIncrement(t *testing.T) {
	ReconcileTotal.WithLabelValues("broker").Inc()
	if got := testutil.ToFloat64(ReconcileTotal.WithLabelValues("broker")); got != 1 {
		t.Errorf("ReconcileTotal{controller=broker} = %v, want 1", got)
	}

	ReconcileErrors.WithLabelValues("meshsync").Inc()
	if got := testutil.ToFloat64(ReconcileErrors.WithLabelValues("meshsync")); got != 1 {
		t.Errorf("ReconcileErrors{controller=meshsync} = %v, want 1", got)
	}
}

// TestReconcileDurationObserve verifies the histogram records observations.
func TestReconcileDurationObserve(t *testing.T) {
	ReconcileDuration.WithLabelValues("broker").Observe(0.5)
	if count := testutil.CollectAndCount(ReconcileDuration); count == 0 {
		t.Error("ReconcileDuration recorded no series after Observe")
	}
}

// TestMetricsRegistered asserts that every metric is registered with the
// controller-runtime registry, which is what exposes them on the operator's
// metrics endpoint.
func TestMetricsRegistered(t *testing.T) {
	// Touch each metric so its family appears in the gathered output.
	ReconcileTotal.WithLabelValues("registered").Inc()
	ReconcileErrors.WithLabelValues("registered").Inc()
	ReconcileDuration.WithLabelValues("registered").Observe(0.1)

	families, err := ctrlmetrics.Registry.Gather()
	if err != nil {
		t.Fatalf("gathering metrics from controller-runtime registry: %v", err)
	}

	want := map[string]bool{
		"meshery_operator_reconcile_total":            false,
		"meshery_operator_reconcile_errors_total":     false,
		"meshery_operator_reconcile_duration_seconds": false,
	}
	for _, family := range families {
		if _, ok := want[family.GetName()]; ok {
			want[family.GetName()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("metric %q is not registered with the controller-runtime registry", name)
		}
	}
}
