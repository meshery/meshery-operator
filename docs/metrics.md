# Metrics

The operator exports Prometheus metrics for its reconciliation loops so that
controller throughput, error rate, and latency are observable in a running
cluster. This document covers what is exported, how it is wired, and how to add a
new metric. It pairs with the [architecture](architecture.md) overview of the two
controllers.

## What is exported

The `pkg/metrics` package defines three metric vectors, each partitioned by a
single `controller` label (`"broker"` or `"meshsync"`):

| Metric | Type | What it measures |
|--------|------|------------------|
| `meshery_operator_reconcile_total` | `CounterVec` | Total reconciliations per controller. |
| `meshery_operator_reconcile_errors_total` | `CounterVec` | Reconciliations that returned a non-nil error, per controller. |
| `meshery_operator_reconcile_duration_seconds` | `HistogramVec` | Reconciliation wall-clock latency per controller (default Prometheus buckets). |

All three carry the `meshery_operator_` prefix and the same `controller` label,
so they line up in queries - e.g. the per-controller error ratio is
`rate(meshery_operator_reconcile_errors_total[5m]) /
rate(meshery_operator_reconcile_total[5m])`, and a latency SLI is
`histogram_quantile(0.99, rate(meshery_operator_reconcile_duration_seconds_bucket[5m]))`.

These sit alongside the controller-runtime built-ins (`controller_runtime_*`,
`workqueue_*`, `rest_client_*`, Go runtime/process metrics) that the manager
already exposes - our metrics describe *our* reconcile bodies, the built-ins
describe the queue and client machinery underneath them.

## How it is wired

There is no manual registration or HTTP wiring to maintain. Two mechanisms do the
work:

1. **Registration** - `pkg/metrics/metrics.go` registers the three vectors with
   the **controller-runtime** registry (`sigs.k8s.io/controller-runtime/pkg/metrics`)
   in its `init()` via `ctrlmetrics.Registry.MustRegister(...)`. That registry is
   the one the manager already serves, so registering there means the metrics
   appear on the existing endpoint automatically.

2. **Exposure** - `cmd/main.go` configures the manager's metrics server
   (`server.Options{BindAddress: metricsAddr}`), which defaults to `:8080` and is
   overridable with `--metrics-addr`. The metrics are scraped from
   `http://<pod>:8080/metrics`.

   > Today this endpoint is plain HTTP (insecure). Metrics TLS + `authn/authz`
   > hardening, and shipping the (currently commented-out) `config/prometheus`
   > `ServiceMonitor`, are tracked under **WS-5** in the
   > [modernization plan](proposals/operator-modernization-plan.md) - until then,
   > scraping is via direct pod access, not a `ServiceMonitor`.

### Instrumentation in the controllers

Each `Reconcile` is instrumented with a single `defer` at the top of the method,
using **named return values** so the deferred closure can read the outcome
without rewriting any of the individual `return` sites:

```go
func (r *BrokerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reconcileErr error) {
    start := time.Now()
    defer func() {
        metrics.ReconcileTotal.WithLabelValues(brokerControllerName).Inc()
        metrics.ReconcileDuration.WithLabelValues(brokerControllerName).Observe(time.Since(start).Seconds())
        if reconcileErr != nil {
            metrics.ReconcileErrors.WithLabelValues(brokerControllerName).Inc()
        }
    }()
    // ... reconcile body, unchanged ...
}
```

Why this shape:

- **One defer, every exit path.** Reconcile has several early returns (not found,
  finalizer requeue, reconcile error). A single deferred closure records the count
  and duration on *all* of them and bumps the error counter only when
  `reconcileErr` is non-nil - no per-return bookkeeping to forget.
- **Named returns are required for it to work.** The closure reads `reconcileErr`
  after the body has set it. Switching to named returns is the only control-flow
  change; it also means an inner `result` variable (e.g. from `ensureFinalizer`)
  must be renamed (`res`) to avoid shadowing the named return - `govet`'s shadow
  analyzer flags this.
- **Controller name is a constant.** `brokerControllerName` / `meshsyncControllerName`
  are declared next to each controller's other constants. They double as the label
  value and satisfy `goconst` (the string would otherwise repeat).

## Adding a metric

1. Define the metric vector in `pkg/metrics/metrics.go` as a package-level `var`,
   with a `meshery_operator_` name prefix and a clear `Help` string. Reuse the
   `controllerLabel` constant if it is per-controller; introduce a new label
   constant if you need a different dimension (keep cardinality bounded - labels
   must be a small, closed set, never user input or resource names).
2. Add it to the `MustRegister(...)` call in `init()`.
3. Record from the relevant code path. For per-reconcile signals, extend the
   existing `defer` closure rather than scattering new call sites.
4. Add a case to `pkg/metrics/metrics_test.go` (see below). New behavior gets a
   test case in the existing suite, not a new file - same convention as the rest
   of the repo ([testing.md](testing.md)).

## Tests

`pkg/metrics/metrics_test.go` uses `client_golang`'s `testutil` helpers and covers
three things:

- **Counters increment** independently per `controller` label
  (`testutil.ToFloat64`).
- **The histogram records** observations (`testutil.CollectAndCount`).
- **Registration** - the acceptance criterion. It gathers from the
  controller-runtime registry (`ctrlmetrics.Registry.Gather()`) and asserts all
  three metric families are present, because registering with *that* registry is
  exactly what exposes them on the operator's endpoint.

These are unit tests with no control plane:

```bash
go test ./pkg/metrics/...
```

## Verifying locally

Run the manager and curl the endpoint:

```bash
make run    # serves metrics on :8080 by default
curl -s localhost:8080/metrics | grep meshery_operator_
```

The series only appear after the corresponding controller has reconciled at least
once (counters and histograms are created lazily on first `WithLabelValues`), so
apply a `Broker`/`MeshSync` sample (or wait for an existing one to reconcile)
before scraping.

## Background

This metrics set fixes [#779](https://github.com/meshery/meshery-operator/issues/779)
and continues the work begun in
[#780](https://github.com/meshery/meshery-operator/pull/780), rebased onto the
post-Kubebuilder-`go/v4` (WS-1) layout. The metric naming and the defer-based
timing pattern carry over from that PR; the Makefile/`CONTRIBUTING.md` edits from
#780 were dropped because WS-1 already provides the envtest-aware `make test`
target and this contributor docs set.
