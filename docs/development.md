# Development

## Prerequisites

- **Go** - the version in [`go.mod`](../go.mod) (`go-version-file` is used in CI).
- **Docker** - for building the manager image and running the kind e2e suite.
- **kubectl**, and a cluster for manual testing (Docker Desktop, kind, or minikube).

All other tooling (`controller-gen`, `kustomize`, `setup-envtest`, `kind`,
`golangci-lint`, `opm`) is installed on demand into `./bin` by the Makefile at
pinned versions - you do not need them on your `PATH`.

## Project layout

The project uses the Kubebuilder **`go.kubebuilder.io/v4`** layout (see
[`PROJECT`](../PROJECT)). The manager entrypoint is **`cmd/main.go`**. Both the
`Broker` and `MeshSync` resources are registered with a controller.

## Common Makefile targets

| Target | What it does |
|--------|--------------|
| `make manifests` | Regenerate CRDs and the RBAC `ClusterRole` from `+kubebuilder` markers via `controller-gen`. |
| `make generate` | Regenerate `zz_generated.deepcopy.go`. |
| `make build` | `go build` the manager into `bin/manager` (from `cmd/main.go`). |
| `make run` | Run the manager against your current kube-context. |
| `make test` | `manifests generate fmt vet` then unit + envtest with `KUBEBUILDER_ASSETS` resolved by `setup-envtest`. |
| `make lint` / `make lint-fix` | Run `golangci-lint` (installed into `bin/`). |
| `make install` / `make deploy` | Apply the CRDs / full operator to the current cluster. |
| `make docker-build IMG=...` | Build the manager image. |
| `make bundle` | Regenerate the OLM bundle (requires `operator-sdk`). |
| `make integration-tests` | Full kind e2e cycle (setup, assert, cleanup). See [testing.md](testing.md). |

After changing API types or `+kubebuilder` markers, always run
`make manifests generate` and commit the regenerated output. CI enforces that the
generated manifests are not stale.

## Tool versions

Tool versions are pinned in the Makefile (`KUSTOMIZE_VERSION`,
`CONTROLLER_TOOLS_VERSION`, `ENVTEST_K8S_VERSION`, `KIND_VERSION`,
`GOLANGCI_LINT_VERSION`, `OPM_VERSION`). The install targets are version-aware:
they reinstall a tool when the on-disk binary reports a different version, so
bumping a pin takes effect on the next `make`.

`ENVTEST_K8S_VERSION` is kept aligned with the `k8s.io/*` library minor version
in `go.mod` so the envtest control plane matches the compiled API surface.

## Building and running locally

```bash
# Run the controllers from your machine against the current kube-context:
make install            # install the CRDs
make run                # run the manager locally

# Or build an image and deploy in-cluster:
make docker-build IMG=meshery/meshery-operator:dev
make deploy IMG=meshery/meshery-operator:dev
```

The manager image is multi-stage and distroless (`gcr.io/distroless/static:nonroot`,
`CGO_ENABLED=0`, `TARGETOS`/`TARGETARCH`), so it builds for both amd64 and arm64.

## Release artifact propagation

The full release flow - what fires on publish, how CRDs/charts sync into
`meshery/meshery`, chart version streams, and the release checklist - is
documented in [release-process.md](release-process.md). The local tooling:

`make crds` renders the two distributable CRD bundle variants into `dist/`
(gitignored):

- **`dist/crds.yaml`** - plain `config/crd/bases` output; conversion strategy
  `None`. This is what the meshery-operator Helm chart ships. It is correct
  **only while the `v1alpha1` and `v1alpha2` schemas are field-identical**
  (the apiserver serves both versions from storage without field mapping).
  When the schemas diverge, the chart must move to webhook conversion - see
  the comment in `api/v1alpha1/conversion.go`.
- **`dist/crds-webhook-conversion.yaml`** - `kustomize build config/crd`; the
  same rendering the operator's own kustomize deployment applies. Conversion
  is wired to the `meshery-webhook-service` Service in the `meshery` namespace
  with cert-manager CA injection, so it requires both.

To dry-run the downstream sync locally against a `meshery/meshery` checkout:

```bash
make crds
hack/sync-downstream.sh ~/code/meshery 1.0.0   # bare version, no leading v
```

The script is idempotent; run it twice and the second run reports
`sync: no changes`.
