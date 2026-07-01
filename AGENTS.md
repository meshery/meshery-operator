# AGENTS.md

This file provides guidance to AI coding agents (including Claude Code) when
working with code in this repository.

## Overview

Meshery Operator is a [Kubebuilder `go/v4`](https://book.kubebuilder.io)
Kubernetes operator. Meshery Server installs one per managed cluster and
thereafter manages its health/config. It owns two CRDs and reconciles each
into concrete workloads:

- **`Broker`** (`brokers.meshery.io`) — a NATS message broker → `StatefulSet`,
  client `Service`, and `nats.conf`/accounts `ConfigMap`s.
- **`MeshSync`** (`meshsyncs.meshery.io`) — the cluster-state synchronizer →
  `Deployment`.

For depth beyond this file, see `docs/architecture.md`, `docs/development.md`,
`docs/testing.md`, `docs/errors.md`, `docs/metrics.md`, and
`docs/proposals/operator-modernization-plan.md` — the active roadmap. Code
comments referencing `WS-N` (e.g. `WS-3`) point at that plan's workstreams.

## Commands

Every build/test/lint workflow is Makefile-driven (`make help` for the full
categorized list). The Makefile installs `controller-gen`, `kustomize`,
`setup-envtest`, `kind`, `golangci-lint`, and `opm` on demand into `./bin` at
pinned versions — none need to be on `PATH`. Go version is pinned in `go.mod`
(CI resolves it via `go-version-file`).

| Command | Purpose |
|---|---|
| `make build` | Compile `cmd/main.go` into `bin/manager`. |
| `make test` | `manifests generate fmt vet`, then unit + envtest suites (auto-resolves `KUBEBUILDER_ASSETS` via `setup-envtest`). |
| `go test ./pkg/...` | Just the fast unit tests (resource builders, pure helpers) — no control plane. |
| `go test ./controllers/... ./pkg/broker/...` | Just the Ginkgo/Gomega envtest suites. |
| `go test ./pkg/broker/... -run TestXxx` | Run a single Go test. |
| `make lint` / `make lint-fix` | Run / auto-fix `golangci-lint`. |
| `make manifests generate` | Regenerate CRDs, the RBAC `ClusterRole`, and `zz_generated.deepcopy.go` from `+kubebuilder` markers. **Required after any API type or marker change** — CI fails on drift. |
| `make nats-manifests` | Re-render `pkg/broker/manifests/nats.gen.yaml` from the pinned official NATS Helm chart (`NATS_CHART_VERSION`) + `pkg/broker/chart/values.yaml`. CI has a drift gate for this too. |
| `make error` | Read-only MeshKit error-registry check (uniqueness, deprecated `NewDefault` usage). Run after touching any `error.go`. |
| `make error-util` | Assign codes to new error placeholders and bump `next_error_code` in `helpers/component_info.json`. |
| `make run` / `make install` / `make deploy` | Run the manager locally against the current kube-context / install CRDs / deploy the operator into the cluster. |
| `make integration-tests` | Full kind e2e cycle: build image, load into kind, deploy, assert Broker/MeshSync become ready. Needs Docker + kind. |
| `make bundle` | Regenerate and validate the OLM bundle (needs `operator-sdk` on `PATH`). |

## Architecture

```
Meshery Server ──(pkg/client/v1alpha1 typed clientset)──▶ Meshery Operator (manager Deployment)
                                                              │
                        ┌─────────────────────────────────────┴───────────────┐
                        ▼                                                       ▼
                 Broker controller                                     MeshSync controller
                        │                                                       ▼
            ┌───────────┼───────────────┐                                 Deployment
            ▼           ▼               ▼                             (BROKER_URL env)
       StatefulSet   Service        2x ConfigMap
        (NATS)       (client)      (nats.conf, accounts)
                        └──▶ endpoint derivation ──▶ Broker.Status.Endpoint ──▶ injected into MeshSync
```

- **`api/v1alpha1/`, `api/v1alpha2/`** — CRD Go types. `v1alpha2` is the
  **conversion hub** (storage version; `Hub()` in `api/v1alpha2/conversion.go`).
  `v1alpha1` is a **spoke** that round-trips to it via JSON
  (`api/v1alpha1/conversion.go`) — the two schemas are currently
  field-identical, so this is a lossless copy; when they diverge, replace the
  round-trip with explicit field mapping. `cmd/main.go` registers the
  conversion webhook for both `Broker` and `MeshSync`; the CRDs declare
  `strategy: Webhook`, so the webhook must be running for either version to be
  served.
- **`controllers/`** — `broker_controller.go` / `meshsync_controller.go`. Both
  follow the same shape: finalizer → main reconcile (create-or-sync owned
  objects) → health check → typed status `Condition` patch via
  `meta.SetStatusCondition`. Each `Reconcile` uses one `defer` with **named
  return values** to record `pkg/metrics` (count/duration/errors) on every exit
  path — see `docs/metrics.md` for why named returns are required.
- **`pkg/broker/`, `pkg/meshsync/`** — resource builders. Workload manifests
  are **hand-authored Go structs**, not Helm/embedded YAML. `pkg/broker/broker.go`
  derives the NATS endpoint and checks health via `ReadyReplicas`;
  `pkg/meshsync/meshsync.go` injects the broker URL and checks health.
- **`pkg/broker/chart/`, `pkg/broker/manifests/nats.gen.yaml`** — the vendored
  NATS topology is rendered at build time from the pinned official `nats/nats`
  Helm chart. Helm is a dev-time-only tool; the operator has no Helm runtime
  dependency and applies the rendered manifest itself. Change NATS topology by
  editing `pkg/broker/chart/values.yaml` (or bumping the chart version), then
  `make nats-manifests`.
- **`pkg/client/v1alpha1/`** — hand-rolled typed clientset (`Brokers(ns)`,
  `MeshSyncs(ns)`) exported for **Meshery Server**, the operator's primary
  consumer. Any `v1alpha1` contract change must preserve this surface.
- **`config/`** — Kustomize bases assembled by `config/default`. **`bundle/`**
  — the OLM bundle, generated from `config/manifests` by `operator-sdk`.

### Known structural debt (tracked in the modernization plan)

- The reconcile sync overwrites `existing.Spec = desired.Spec` and gates on
  `Semantic.DeepEqual`, which fights server-defaulted fields and can hot-loop.
  Server-Side Apply replaces this in WS-3 (#785).
- Only the primary workload is `Owns()`'d; the broker `Service`/`ConfigMap`s
  are not watched, so endpoint changes don't re-trigger reconciliation (WS-3).
- Endpoint derivation performs blocking TCP dials in the reconcile path; WS-3/
  WS-4 make it a pure, non-blocking function over the `Service`.

### Error handling (required convention)

All errors are **MeshKit structured errors**
(`github.com/meshery/meshkit/errors`), never `fmt.Errorf`/`errors.New`. One
exported code constant (matching `^Err[A-Z].+Code$`) plus one constructor per
error; codes and names are unique across the *whole component*, not just the
package:

```go
const ErrReconcileBrokerCode = "1006"

func ErrReconcileBroker(err error) error {
    return meshkiterrors.New(ErrReconcileBrokerCode, meshkiterrors.Alert,
        []string{"Broker reconciliation failed"}, []string{err.Error()},
        []string{"probable cause"}, []string{"suggested remediation"})
}
```

- ShortDescription/ProbableCause/SuggestedRemediation must be string literals
  (the errorutil tool extracts them for the error reference); the dynamic
  cause (`err.Error()`) goes only in LongDescription.
- Include the offending resource's name/namespace in the description — say
  *which* `Broker`/`MeshSync` failed, not just "configuration invalid".
- Surface failures to the CR's status `Condition` (and `PublishingTo`/
  `Endpoint` where relevant), not only logs.
- Allocate the next code from `helpers/component_info.json`'s
  `next_error_code`; run `make error` after any change. Current registries:
  `controllers/error.go` (1001–1012, 1017), `pkg/broker/error.go`
  (1013–1016), `pkg/meshsync/error.go` (1018–1021).

### Testing

Three tiers (`docs/testing.md`):

1. **Unit** (`go test ./pkg/...`) — table tests for resource builders and pure
   helpers, no control plane.
2. **envtest** (`make test`) — Ginkgo/Gomega suites in `controllers/` and
   `pkg/broker/` reconciling against a real `kube-apiserver`+`etcd`, with no
   kubelet. Pods never actually run, so tests needing health/endpoint behavior
   drive the workload `.status` subresource directly. Any test object still
   needs to pass real API validation (e.g. a `StatefulSet`/`Deployment` pod
   template needs at least one container or the apiserver rejects it with a
   422).
3. **kind e2e** (`make integration-tests`) — builds the image, loads it into
   kind, deploys via `config/default`, applies the `Broker`/`MeshSync`
   samples, and asserts the broker and meshsync workloads become ready.

New behavior gets a case added to the existing suite file for its package,
not a new test file.

## Commit conventions

Do not add AI attribution to commit messages, PR descriptions, or code
comments — no "Co-Authored-By" trailers naming an AI vendor, no "Generated
with/by" boilerplate naming an AI tool, and no links to an AI vendor's share
domains. A local hook in this repo blocks any Bash command (including `git
commit`) or file write matching these patterns.
