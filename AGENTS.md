# AGENTS.md

This file provides guidance to AI coding agents (including Claude Code) when working with code in this repository. It is an index; depth lives in [docs/](docs/README.md).

## Overview

Meshery Operator is a [Kubebuilder `go/v4`](https://book.kubebuilder.io) Kubernetes
operator. Meshery Server installs one per managed cluster and thereafter manages its
health/config. It owns two CRDs and reconciles each into concrete workloads:

- **`Broker`** (`brokers.meshery.io`) - a NATS message broker → `StatefulSet`, client `Service`, and `nats.conf`/accounts `ConfigMap`s.
- **`MeshSync`** (`meshsyncs.meshery.io`) - the cluster-state synchronizer → `Deployment`.

## Commands

Every build/test/lint workflow is Makefile-driven (`make help` for the full
categorized list). The Makefile installs `controller-gen`, `kustomize`, `setup-envtest`,
`kind`, `golangci-lint`, and `opm` on demand into `./bin` at pinned versions - none need
to be on `PATH`. Go version is pinned in `go.mod` (CI resolves it via `go-version-file`).

| Command | Purpose |
|---|---|
| `make build` | Compile `cmd/main.go` into `bin/manager`. |
| `make test` | `manifests generate fmt vet`, then unit + envtest suites (auto-resolves `KUBEBUILDER_ASSETS` via `setup-envtest`). |
| `go test ./pkg/...` | Just the fast unit tests (resource builders, pure helpers) - no control plane. |
| `go test ./controllers/... ./pkg/broker/...` | Just the Ginkgo/Gomega envtest suites. |
| `go test ./pkg/broker/... -run TestXxx` | Run a single Go test. |
| `make lint` / `make lint-fix` | Run / auto-fix `golangci-lint`. |
| `make manifests generate` | Regenerate CRDs, the RBAC `ClusterRole`, and `zz_generated.deepcopy.go` from `+kubebuilder` markers. **Required after any API type or marker change** - CI fails on drift. |
| `make nats-manifests` | Re-render `pkg/broker/manifests/nats.gen.yaml` from the pinned official NATS Helm chart (`NATS_CHART_VERSION`) + `pkg/broker/chart/values.yaml`. CI has a drift gate for this too. |
| `make error` | Read-only MeshKit error-registry check (uniqueness, deprecated `NewDefault` usage). Run after touching any `error.go`. |
| `make error-util` | Assign codes to new error placeholders and bump `next_error_code` in `helpers/component_info.json`. |
| `make run` / `make install` / `make deploy` | Run the manager locally against the current kube-context / install CRDs / deploy the operator into the cluster. |
| `make integration-tests` | Full kind e2e cycle: build image, load into kind, deploy, assert Broker/MeshSync become ready. Needs Docker + kind. |
| `make bundle` | Regenerate and validate the OLM bundle (needs `operator-sdk` on `PATH`). |

## Architecture

Component map - full detail in [docs/architecture.md](docs/architecture.md) and the [modernization plan](docs/proposals/operator-modernization-plan.md):

- `api/v1alpha1/`, `api/v1alpha2/` - CRD Go types; `v1alpha2` is the conversion hub (storage version), `v1alpha1` a spoke served via the conversion webhook.
- `controllers/` - Broker/MeshSync reconcilers: finalizer → create-or-sync owned objects → health check → status `Condition` patch; `pkg/metrics` recorded via named returns.
- `pkg/broker/` + `pkg/meshsync/` - hand-authored resource builders; the NATS topology is rendered from the vendored chart (`make nats-manifests`).
- `pkg/client/v1alpha1/` - typed clientset consumed by Meshery Server; any `v1alpha1` change must preserve this surface.
- `config/` - Kustomize bases; `bundle/` - the OLM bundle generated from `config/manifests`.

## Error handling

All errors are MeshKit structured errors (`github.com/meshery/meshkit/errors`), never
`fmt.Errorf`/`errors.New`: one exported `Err...Code` constant plus one constructor per
error, with codes unique across the whole component and allocated from
`helpers/component_info.json`. Run `make error` after touching any `error.go`.
Full convention and example: [docs/errors.md](docs/errors.md).

## Testing

Three tiers: unit (`go test ./pkg/...`), envtest (`make test`), and kind e2e
(`make integration-tests`). New behavior gets a case added to the existing suite file
for its package, not a new test file. envtest caveats and e2e knobs: [docs/testing.md](docs/testing.md).

## Identifier Naming Conventions

**Wire is camelCase everywhere; DB is snake_case; Go fields follow Go idiom; the ORM layer is the sole translation boundary.**

- Authoritative source: `meshery/schemas/AGENTS.md § Casing rules at a glance`
- Reader-friendly directory: <https://github.com/meshery/schemas/blob/master/docs/identifier-naming-contributor-guide.md>
- The contract is not optional; deviations block PRs via the schemas consumer-audit CI gate. On any conflict, schemas wins - file discrepancies as issues against `meshery/schemas`, not locally.
- `Id` (camelCase), never `ID`, in URL params, JSON tags, and TypeScript properties.
- meshery-operator: Go types follow Go idiom; CRD serialized field names are camelCase per Kubernetes API conventions, which coincides with the ecosystem wire contract. The consumer audit applies wherever operator types cross the wire to Meshery Server.

## Required on Every PR

- **Tests accompany every behavioral change.** Run every locally-runnable test before
  requesting review; never defer runnable coverage to reviewers or follow-up PRs.
- **Documentation accompanies every behavioral change, in both forms:**
  - External, user-facing: docs.meshery.io (source: meshery/meshery docs) - update whenever the change is user-visible.
  - Internal, developer-facing: this repo's [`docs/`](docs/) - update whenever architecture, workflows, or contracts change.
- **Schema-aware changes**: run `cd ../schemas && make validate-schemas && make consumer-audit` before pushing.
- **Sign off every commit** (`git commit -s`).
- **No AI attribution** in commits, PR descriptions, comments, or code.

No AI attribution means: no "Co-Authored-By" trailers naming an AI vendor, no
"Generated with/by" boilerplate naming an AI tool, and no links to an AI vendor's share
domains. A local hook in this repo blocks any Bash command (including `git commit`) or
file write matching these patterns.

## Detailed documentation

- [docs/architecture.md](docs/architecture.md) - CRDs, conversion hub/webhook, controller shape, resource builders, vendored NATS chart, typed client, packaging, known structural debt. Read before structural changes.
- [docs/development.md](docs/development.md) - local setup, Makefile targets, tool pinning, CRD release artifacts.
- [docs/testing.md](docs/testing.md) - the three test tiers in detail, envtest caveats, kind e2e environment knobs.
- [docs/errors.md](docs/errors.md) - the full MeshKit error convention, example constructor, and per-file code registries.
- [docs/metrics.md](docs/metrics.md) - reconciliation metrics and why named returns are required.
- [docs/release-process.md](docs/release-process.md) - release flow and downstream chart/CRD sync into `meshery/meshery`.
- [docs/proposals/operator-modernization-plan.md](docs/proposals/operator-modernization-plan.md) - the active roadmap; code comments referencing `WS-N` (e.g. `WS-3`) point at that plan's workstreams.
