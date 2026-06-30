# Meshery Operator Modernization Plan

**Status:** Proposed
**Target repository:** `meshery/meshery-operator`
**Scope:** End‑to‑end overhaul of the operator — scaffolding/SDK, API/CRDs, custom controllers, the NATS broker provisioning and its post‑deployment service networking, RBAC, dependencies, lifecycle testing, CI/CD, and OLM packaging.
**Related repositories:** `meshery/meshkit` (endpoint derivation, error registry), `meshery/meshsync` (the workload the operator deploys), `meshery/meshery` (Meshery Server, the operator's primary consumer/manager).

---

## 1. Executive summary

Meshery Server automatically installs Meshery Operator into a Kubernetes cluster on connection, and thereafter manages the operator's health and configuration. The operator owns two CRDs — `Broker` (a NATS message broker) and `MeshSync` (a cluster‑state synchronizer) — and reconciles each into concrete workloads (a `StatefulSet`/`Service`/`ConfigMap`s for NATS; a `Deployment` for MeshSync).

The codebase has received **genuine, recent investment**: finalizers, typed status `Conditions`, a custom‑broker handler, "reconcile existing owned resources on update", and Ginkgo/Gomega `envtest` suites have all landed in the last several change sets, and `go.mod` was recently bumped to Go 1.26, Kubernetes `v0.35.0` libraries, and controller‑runtime `v0.22.4`. This plan is **not** a rewrite‑from‑scratch argument; it is a roadmap that consolidates that progress and closes the structural gaps that piecemeal fixes have left behind.

Those gaps are significant and, in several cases, production‑affecting:

- **The project scaffolding is frozen in 2021–2022.** `PROJECT` still declares `layout: go.kubebuilder.io/v2` with an `*-v2-alpha` plugin; the OLM bundle is stamped `operator-sdk-v1.14.0` / `createdAt: 2022‑05‑12` and has never moved off version `0.0.1`. The modern, recently‑bumped *dependencies* sit on top of a *project layout* that predates them by four years.
- **The operator runs with effective cluster‑admin.** `config/rbac/controller_role.yaml` grants `apiGroups: ['*'], resources: ['*']` for create/delete/get/list/patch/update/watch, bound to the operator's ServiceAccount. The auto‑generated least‑privilege role is simultaneously *incomplete* (it omits the `apps`/core resources the controllers actually manage), so the wildcard role is silently load‑bearing.
- **NATS service networking cannot be reconfigured.** `BrokerSpec` exposes a single field, `Size`. The Service `Type` is hard‑coded to `LoadBalancer`, all six ports are hard‑coded, and the external endpoint is derived by a snapshot helper that performs **blocking TCP dials inside the reconcile loop** and is never re‑triggered by Service changes (the controller does not watch the Service). This is the explicit motivating concern of this plan and is treated in depth in §6.
- **The manager Deployment is unsafe and unreliable** — `hostPort` bindings, no `securityContext`, no liveness/readiness probes, and a `30Mi` memory limit that will OOMKill a controller‑runtime manager carrying meshkit's large transitive dependency tree.
- **Lifecycle testability is shallow.** `envtest` validates object creation but cannot validate health/endpoint behavior (no kubelet → pods never become Ready), and its asset path is hard‑pinned to `bin/k8s/1.30.0-linux-amd64`, breaking on arm64/macOS — ironic given the repo just added an ARM multi‑arch build.
- **Dependency bloat and version skew.** The operator pins `meshkit v0.8.64` while the rest of the ecosystem has moved to `meshkit v1.0.x` (MeshSync uses `v1.0.4`), and through meshkit the operator inherits Helm, GORM, SQLite, Postgres, CUE, and ORAS for what amounts to a single endpoint helper.

The plan is organized into **eight workstreams** delivered across **five phases** (§5), with the NATS networking reconfiguration (§6) and lifecycle testing (§7) called out as first‑class deliverables because they are the named objectives. Each workstream lists concrete findings (with `file:line` references), a target state, and acceptance criteria.

---

## 2. Assessment methodology

This plan is grounded in a direct read of the repository at branch tip, a build/verify pass, and a cross‑repository read of `meshkit` and `meshsync`. Specifically:

- **Static read** of every Go source file under `api/`, `controllers/`, `pkg/`, plus `main.go` (~4,300 LoC total).
- **Build verification**: `go build ./...` succeeds on Go 1.26.4; `go vet ./...` and `go list -deps ./...` exceed two minutes — itself a signal of the transitive dependency weight pulled in through meshkit.
- **Manifest/scaffolding read** of `config/**`, `bundle/**`, `Dockerfile`, `Makefile`, `PROJECT`, `.golangci.yml`, `.github/workflows/**`, and `integration-tests/**`.
- **Cross‑repo read** of meshkit's `utils/kubernetes/service.go` (`GetEndpoint`) and meshsync's broker/NATS configuration path, to establish the contract the operator must preserve.
- **Version grounding** against current upstream releases (operator‑sdk v1.42.2, Kubebuilder `go/v4`, the official NATS Helm chart line, controller‑tools).

Where a claim is load‑bearing (e.g., the wildcard RBAC, the cert‑manager API version, the frozen bundle), it was verified by reading the exact file rather than inferred.

---

## 3. Current‑state architecture

```
Meshery Server  ──(installs & manages)──▶  Meshery Operator (manager Deployment)
                                              │
                  ┌───────────────────────────┴───────────────────────────┐
                  ▼                                                         ▼
          Broker controller                                        MeshSync controller
          reconciles Broker CR                                     reconciles MeshSync CR
                  │                                                         │
       ┌──────────┼───────────────┐                                        ▼
       ▼          ▼               ▼                                  Deployment
  StatefulSet  Service(LB)   2× ConfigMap                          meshery/meshsync
   nats:2.8.2  6 ports        nats.conf + accounts                 BROKER_URL env
                  │
                  └──▶ meshkit GetEndpoint() ──▶ Broker.Status.Endpoint.{Internal,External}
                                                          │
                                                          └──▶ injected as MeshSync BROKER_URL
```

Key facts that shape the plan:

- **Two `v1alpha1` CRDs**, `meshery.io/Broker` and `meshery.io/MeshSync`, namespaced, `apiextensions.k8s.io/v1` (good), generated by controller‑gen `v0.17.1`.
- **Two reconcilers** (`controllers/broker_controller.go`, `controllers/meshsync_controller.go`) wired in `main.go`. Both now implement finalizers, status conditions, and an "update existing owned object" path.
- **Resource templates are hand‑authored Go structs** in `pkg/broker/resources.go` and `pkg/meshsync/resources.go` (no Helm/embedded YAML).
- **A hand‑rolled typed clientset** (`pkg/client/v1alpha1/`) is exported for Meshery Server to create/manage `Broker`/`MeshSync` objects.
- **Endpoint derivation is delegated to meshkit** (`meshkitkube.GetEndpoint`), which is the seam where NATS networking is computed.

---

## 4. Detailed findings

Severity legend: **[C]** critical · **[H]** high · **[M]** medium · **[L]** low. Each finding maps to a workstream in §5.

### 4.1 Scaffolding, SDK & framework (WS‑1)

| # | Sev | Finding | Evidence |
|---|-----|---------|----------|
| 1 | H | Project layout is Kubebuilder **v2** with an alpha SDK plugin; current is `go.kubebuilder.io/v4`. The `MeshSync` resource still has `# controller: true` commented out with a scaffold TODO, and `Broker` is not listed as a resource at all. | `PROJECT:1-17` |
| 2 | H | OLM bundle is frozen: `operator-sdk-v1.14.0+git`, `project_layout: go.kubebuilder.io/v2`, `createdAt: "2022-05-12"`, name `meshery-operator.v0.0.1`, version never bumped past `0.0.1`, channel `alpha`, only `AllNamespaces` install mode, **no `replaces`/`skipRange`** (no upgrade graph). Maintainer email is a typo: `urakiny@gmai.com`. | `bundle/0.0.1/metadata/annotations.yaml`, `bundle/0.0.1/manifests/*.clusterserviceversion.yaml:54-596` |
| 3 | M | Makefile carries dead/legacy options: `CRD_OPTIONS ?= "crd:trivialVersions=true"` (the flag was removed from controller‑gen and is not even referenced by the `manifests` target); `KUSTOMIZE_VERSION v3.8.7` (current line is v5); `opm v1.23.0`; `kind v0.18.0`. | `Makefile:56,193,231,271` |
| 4 | M | `make bundle-build` runs `docker build -f bundle.Dockerfile` but **`bundle.Dockerfile` does not exist** in the repo — the target is broken. | `Makefile:216` |
| 5 | M | `ENVTEST_K8S_VERSION = 1.30.0` while the Kubernetes libraries are `v0.35.0` — the test control plane lags the compiled API surface by five minor versions. | `Makefile:11` vs `go.mod:10-13` |
| 6 | L | `lint` target assumes `golangci-lint` is already on `PATH` (no install target); `make run` runs `go mod tidy` as a side effect of every run. | `Makefile:110-112,129-131` |

### 4.2 API / CRD design (WS‑2)

| # | Sev | Finding | Evidence |
|---|-----|---------|----------|
| 7 | H | **`BrokerSpec` has only `Size int32`.** There is no API surface for service type, ports, annotations, external‑access policy, NATS version, or JetStream — so NATS networking simply cannot be expressed declaratively. | `api/v1alpha1/broker_types.go:21-26` |
| 8 | M | `MeshSyncSpec.WatchList` embeds a **full `corev1.ConfigMap`** (TypeMeta + ObjectMeta + Data + BinaryData + Status) directly into the CRD schema — a heavyweight anti‑pattern that bloats the OpenAPI schema and stored objects. | `api/v1alpha1/meshsync_types.go:39-41` |
| 9 | M | `MeshSyncSpec.Version` is **unused** — the Deployment image is always `meshery/meshsync:stable-latest`, so the declared version is silently ignored. | `meshsync_types.go:42` vs `pkg/meshsync/resources.go:89` |
| 10 | M | Both CRDs are still `v1alpha1` after years in production, with **no `v1beta1`/`v1`**, no conversion strategy in force, no defaulting/validation webhooks active, no `additionalPrinterColumns`, `shortNames`, `categories`, or CEL (`x-kubernetes-validations`) rules. | `api/v1alpha1/*_types.go`, `config/crd/bases/*` |
| 11 | L | `Broker` struct declares `Status` before `Spec`, contrary to convention. | `broker_types.go:42-47` |

### 4.3 Controller correctness & robustness (WS‑3)

| # | Sev | Finding | Evidence |
|---|-----|---------|----------|
| 12 | C | **Leader election is non‑functional.** `LeaderElectionID` is built from a fresh random UUID per process: `fmt.Sprintf("operator-%s.meshery.io", uuid.NewUUID())`. Every pod/restart picks a unique lease name, so replicas never contend — the `--enable-leader-election` flag passed in the manifest is inert. | `main.go:60,70` |
| 13 | H | **Reconcile hot‑loop risk.** The sync helpers overwrite `existing.Spec = desired.Spec` and gate on `Semantic.DeepEqual(existing.Spec, desired.Spec)`. Server‑defaulted fields (Service `clusterIP`, per‑port `nodePort`, `healthCheckNodePort` on `LoadBalancer`; StatefulSet/Deployment defaults) will perpetually differ from the bare desired spec, producing endless no‑op `Update`s. Worst on the `LoadBalancer` Service. | `broker_controller.go:355-422`, `meshsync_controller.go:325-342` |
| 14 | H | **Blocking network I/O in the reconcile path.** Endpoint derivation calls meshkit `GetEndpoint`, which performs up to two `net.DialTimeout("tcp", …, 5s)` probes synchronously. A single reconcile can block 5–15s, starving the controller work queue. | `pkg/broker/broker.go:88-117`; meshkit `utils/kubernetes/service.go` + `utils/network.go` |
| 15 | H | **No Service/ConfigMap watch.** Broker `SetupWithManager` only `Owns(&appsv1.StatefulSet{})`; MeshSync only `Owns(&appsv1.Deployment{})`. Changes to the NATS `Service` (type/ports/LB IP assignment) do not trigger reconciliation, so the derived endpoint is never recomputed when networking changes. | `broker_controller.go:258-263`, `meshsync_controller.go:83-88` |
| 16 | M | `GetObjects` returns a **`map[string]Object`** (non‑deterministic iteration), and `reconcileBroker` returns immediately after the first `Create`. Creation order across StatefulSet/Service/ConfigMaps is random and spread across multiple reconcile passes; only the StatefulSet is watched, so progress relies on requeues. | `pkg/broker/broker.go:28-35`, `broker_controller.go:286-328` |
| 17 | M | `CheckHealth` inspects `Status.Conditions[0]` of a StatefulSet/Deployment; StatefulSets do not populate `conditions`, so the real signal is `ReadyReplicas`. The condition branch is effectively dead and order‑fragile. | `pkg/broker/broker.go:66-85`, `pkg/meshsync/meshsync.go:56-75` |
| 18 | M | MeshSync's broker URL is injected positionally as `Env[0].Value` and sourced from `Status.PublishingTo`, which is the **Internal (ClusterIP)** address with **no `nats://` scheme**, and is `""` when no broker reference is set. The hard‑coded template default `http://localhost:4222` (wrong host *and* wrong scheme) is therefore dead/misleading. | `pkg/meshsync/meshsync.go:46-54`, `pkg/meshsync/resources.go:101-106`, `meshsync_controller.go:265-282` |
| 19 | M | `main.go` imports the deprecated in‑tree `k8s.io/client-go/plugin/pkg/client/auth/gcp` (still compiles in client‑go v0.35, but GKE moved to the `gke-gcloud-auth-plugin` exec credential model years ago). No `healthz`/`readyz` checks are registered; metrics serve on insecure `:8080`. | `main.go:31,49-72` |
| 20 | L | Error handling uses `fmt.Errorf` with string codes (`1001`–`1012`) that are only partially aligned with the meshkit error registry (`helpers/component_info.json` → `next_error_code: 1017`); `ErrMarshalCode = "11049"` is an outlier, and meshkit's structured `errors.New(code, severity, …, probableCause, remedy)` framework is not used. A stray `// @Aisuko …` review note is committed. | `controllers/error.go:23-91` |
| 21 | L | The hand‑rolled typed clientset stores `ParameterCodec` as a **mutable package global** set in `New()` (not thread‑safe), and re‑implements what `k8s.io/code-generator` or the controller‑runtime client already provide. | `pkg/client/v1alpha1/v1alpha1.go:8-27` |

### 4.4 NATS broker provisioning & networking (WS‑4 — see §6)

| # | Sev | Finding | Evidence |
|---|-----|---------|----------|
| 22 | C | **Service `Type: LoadBalancer` is hard‑coded** with all six ports fixed. On bare‑metal/kind/minikube (no cloud LB controller) the Service stays `<pending>` and external endpoint derivation degrades to brittle fallbacks. No CRD field overrides this. | `pkg/broker/resources.go:87-124` |
| 23 | H | **NATS is ~4 years stale.** Server image `nats:2.8.2-alpine3.15` (May 2022; current line is 2.10.x/2.11.x). The config‑reloader is `connecteverything/nats-server-config-reloader:0.6.0` — `connecteverything` is the **defunct** NATS org (now `natsio`). The whole hand‑authored StatefulSet predates the official NATS Helm chart. | `pkg/broker/resources.go:185-296` |
| 24 | H | **Endpoint derivation has no post‑deploy story.** meshkit `GetEndpoint` is a point‑in‑time snapshot using heuristics (string‑comparing the LB ingress IP to the literal `"<pending>"`, minikube API‑server host fallback, `WorkerNodeIP` defaulting to `localhost`) and blocking TCP checks. Combined with finding #15 (no Service watch), the operator cannot react to a Service type change or a delayed LB IP assignment. | `pkg/broker/broker.go:88-117`; meshkit `service.go` |
| 25 | M | A **NATS account JWT is committed** in the accounts ConfigMap (a `resolver: MEMORY` preload), and `resolver: MEMORY` is declared twice (in `nats.conf` and `resolver.conf`). | `pkg/broker/resources.go:60-85` |

### 4.5 RBAC & security posture (WS‑5)

| # | Sev | Finding | Evidence |
|---|-----|---------|----------|
| 26 | C | **Effective cluster‑admin.** `controller-role` grants `apiGroups: ['*'], resources: ['*']` × {create,delete,get,list,patch,update,watch}, bound to the `meshery-operator` ServiceAccount. | `config/rbac/controller_role.yaml`, `controller_role_binding.yaml` |
| 27 | H | The **generated least‑privilege role is incomplete**: `operator-role` covers only `meshery.io/{brokers,meshsyncs}` (+status), but the controllers create/update/delete `apps/{statefulsets,deployments}` and core `{services,configmaps}`. The `+kubebuilder:rbac` markers omit those groups, so `make manifests` cannot produce a correct role — the wildcard role (#26) silently compensates. | `config/rbac/role.yaml`, `controllers/*_controller.go:55-56` |
| 28 | H | Manager Deployment uses **`hostPort: 9443` and `hostPort: 8080`** (node‑level binding → one operator per node, direct node exposure), defines **no `securityContext`** (no `runAsNonRoot`, `readOnlyRootFilesystem`, `drop: [ALL]`, `seccompProfile`), has **no liveness/readiness probes**, and sets a **`30Mi` memory limit** that will OOMKill the manager. Image is `stable-latest` with `imagePullPolicy: Always`. | `config/manager/manager.yaml:32-53` |
| 29 | M | Deprecated kube‑rbac‑proxy scaffolding (`auth_proxy_*`) lingers; operator‑sdk/Kubebuilder have moved metrics protection to cert‑manager‑issued TLS + `authn/authz` filters. cert‑manager `Certificate` uses the removed `cert-manager.io/v1alpha2` API (dormant, since webhooks are disabled — see #30). | `config/rbac/auth_proxy_*.yaml`, `config/certmanager/certificate.yaml:5,13` |
| 30 | M | The webhook + conversion + prometheus scaffolding is **vestigial**: `config/default/kustomization.yaml` comments out `../webhook`, `../certmanager`, and `../prometheus`, so no admission/conversion webhooks or ServiceMonitor ship. | `config/default/kustomization.yaml:19-56` |
| 31 | L | `.golangci.yml` enables a reasonable set (`govet`, `staticcheck`, `gocritic`, `cyclop`, …) but **no `gosec`** security linter. | `.golangci.yml` |

### 4.6 Dependencies (WS‑6)

| # | Sev | Finding | Evidence |
|---|-----|---------|----------|
| 32 | H | **meshkit is a major version behind.** The operator pins `meshkit v0.8.64`; MeshSync already uses `meshkit v1.0.4`. The shared library has shipped a `v1.0.x` line the operator has not adopted. | `go.mod:7`; meshsync `go.mod` |
| 33 | H | **Dependency bloat.** Through meshkit, the operator's module graph pulls Helm v3, GORM (+ SQLite, Postgres drivers), CUE, ORAS, and nats.go — for what is, in practice, a single endpoint helper plus a few error constructors. This inflates the binary, slows tooling (`go vet`/`go list` >2 min), and widens the supply‑chain surface. | `go.mod:17-181` |
| 34 | M | **client‑go skew across the ecosystem**: operator `v0.35.0`, meshsync `v0.35.3`, meshkit `v0.34.3`. Divergent minor/patch versions complicate shared types and transitive resolution. | respective `go.mod`s |

### 4.7 Lifecycle testing (WS‑7 — see §7)

| # | Sev | Finding | Evidence |
|---|-----|---------|----------|
| 35 | H | `envtest` provides only apiserver+etcd (no kubelet), so Pods never become Ready and `ReadyReplicas`‑based `CheckHealth` can never pass — **health and endpoint behavior are untestable** at the unit/integration tier; only object creation is covered. | `controllers/suit_test.go:56-154` |
| 36 | H | `BinaryAssetsDirectory` is hard‑pinned to `bin/k8s/1.30.0-linux-amd64` — **breaks on arm64/macOS**, directly conflicting with the newly added ARM multi‑arch workflow. Should resolve via `setup-envtest`/`KUBEBUILDER_ASSETS`. | `suit_test.go:69` |
| 37 | M | Test‑harness defects: unchecked error on `clientSet` init; the MeshSync reconciler logger is mislabeled `"Broker"`; a `// isten…` typo. | `suit_test.go:110,127,103` |
| 38 | M | Integration tests are bash on kind with **no pinned Kubernetes version** (cluster version drifts between runs) and 300s polling. They assert MeshSync Deployment readiness, Broker StatefulSet `readyReplicas=1`, and that the Broker CR status `endpoint.{external,internal}` populate — but there is **no test for networking reconfiguration**, no NodePort/ClusterIP matrix, and no upgrade/failure scenarios. | `integration-tests/main.sh` |

### 4.8 CI/CD & supply chain (WS‑8)

| # | Sev | Finding | Evidence |
|---|-----|---------|----------|
| 39 | M | Unpinned/abandoned GitHub Actions: `actions/checkout@master` (in `build-and-release.yml`, `label-commenter.yml`), `azure/docker-login@v1` (superseded by `docker/login-action`), `pullreminders/slack-action@master` (unmaintained); mixed `ubuntu-22.04`/`24.04`/`latest` runners. | `.github/workflows/*` |
| 40 | M | No security scanning in CI: no CodeQL, no image vulnerability scan (Trivy/Grype), no SBOM, no image signing (cosign). | `.github/workflows/*` |
| 41 | L | Positives to preserve: `multi-platform.yml` (amd64+arm64 via buildx with current `docker/*` actions), `integration-tests-ci.yml` (kind), `release-drafter`, meshkit error‑ref publisher, and `setup-go@v5` with `go-version-file`. The Dockerfile is already modern (multi‑stage, `gcr.io/distroless/static:nonroot`, `CGO_ENABLED=0`, `TARGETOS/TARGETARCH`). | `.github/workflows/*`, `Dockerfile` |

---

## 5. Modernization workstreams & phased roadmap

Eight workstreams, sequenced into five phases. Phases are ordered so that **safety‑critical and enabling** work (RBAC, leader election, test harness) precedes **behavioral** changes (API evolution, NATS networking), which precede **packaging/polish**.

### Workstream summary

| WS | Title | Primary outcomes |
|----|-------|------------------|
| WS‑1 | SDK & scaffolding | Migrate to Kubebuilder `go/v4` layout; current operator‑sdk; regenerate bundle; fix Makefile. |
| WS‑2 | API & CRD evolution | Introduce `v1alpha2`→`v1`; networking‑capable `BrokerSpec`; trim `MeshSync` spec; printer columns + CEL. |
| WS‑3 | Controller correctness | Fix leader election; SSA‑based reconcile (no hot‑loop); watch Services; non‑blocking endpoint logic; probes. |
| WS‑4 | NATS broker modernization | Upgrade NATS; adopt official chart patterns/JetStream‑ready; **post‑deploy networking reconfiguration** (§6). |
| WS‑5 | RBAC & security hardening | Least‑privilege RBAC; harden manager Deployment; cert‑manager v1; metrics TLS; gosec. |
| WS‑6 | Dependency slimming | Adopt meshkit v1.x or extract the endpoint helper; cut Helm/GORM/CUE from the operator graph; align client‑go. |
| WS‑7 | Lifecycle testing | Robust envtest; e2e on kind with a real NATS; networking‑reconfig and upgrade suites. |
| WS‑8 | CI/CD & supply chain | Pin actions; add CodeQL/Trivy/SBOM/cosign; CRD/bundle drift gates. |

### Phase 0 — Stabilize & de‑risk (safety fixes, no API change)

Goal: eliminate the correctness/security landmines without changing the public CRD contract, so the rest of the work proceeds on a safe base.

- **WS‑3 / #12** Fix leader election: derive `LeaderElectionID` from a **stable** identifier (e.g. `meshery-operator-leader.meshery.io`), not a per‑process UUID. Add `healthz`/`readyz` and `mgr.AddHealthzCheck`/`AddReadyzCheck`.
- **WS‑5 / #26‑28** Replace the wildcard `controller-role` with a least‑privilege ClusterRole and add the missing `+kubebuilder:rbac` markers for `apps/{statefulsets,deployments}` and core `{services,configmaps,configmaps/status}` so `make manifests` regenerates a correct role. Harden `config/manager/manager.yaml`: drop `hostPort`, add `securityContext` (runAsNonRoot, readOnlyRootFilesystem, drop ALL caps, seccomp `RuntimeDefault`), add probes, raise memory limit to a realistic floor (≥128Mi request / 256Mi limit, to be tuned via load test).
- **WS‑3 / #14, #19** Make endpoint derivation **non‑blocking**: move TCP reachability out of the reconcile path (or gate it behind a short context deadline and a requeue), and remove the deprecated `auth/gcp` import.
- **WS‑7 / #36‑37** Fix the test harness: resolve envtest assets via `setup-envtest` (`KUBEBUILDER_ASSETS`), fix the unchecked error and the mislabeled logger.
- **WS‑1 / #4** Either add a real `bundle.Dockerfile` or remove the broken `make bundle-build` reference.

**Acceptance:** leader election demonstrably elects a single leader across 2 replicas; operator runs under a least‑privilege role in a kind cluster with no RBAC `forbidden` errors; manager pod passes probes and survives a soak without OOM; `make test` runs on arm64 and amd64.

### Phase 1 — Tooling & scaffolding refresh (WS‑1, WS‑8)

- Migrate `PROJECT` to `layout: go.kubebuilder.io/v4`; register **both** `Broker` and `MeshSync` resources with `controller: true`. Re‑scaffold the manager wiring (`cmd/main.go`, metrics TLS, `--health-probe-bind-address`) to the v4 conventions, preserving existing controller logic.
- Refresh the Makefile: remove `trivialVersions`; bump `KUSTOMIZE`, `controller-gen`, `setup-envtest`, `opm`, `kind`; align `ENVTEST_K8S_VERSION` with the compiled k8s libraries; add a `golangci-lint` install target.
- CI hardening (WS‑8): pin all actions to immutable versions/SHAs; standardize on `ubuntu-24.04`; add CodeQL, Trivy image scan, SBOM (`syft`), and cosign signing; add a "CRD/RBAC/bundle drift" gate (`make manifests bundle && git diff --exit-code`).

**Acceptance:** `operator-sdk`/Kubebuilder `v4` `make` targets succeed; `git diff` is clean after regeneration; CI runs CodeQL + image scan + drift gate on every PR.

### Phase 2 — API evolution (WS‑2) with a networking‑capable Broker

- Introduce **`v1alpha2`** for both kinds (storage = `v1alpha2`, served `v1alpha1` retained), with a conversion webhook so existing `v1alpha1` objects (and Meshery Server's typed client) keep working. Enable the dormant webhook/cert‑manager wiring (cert‑manager `v1` API).
- Evolve `BrokerSpec` to express networking (see §6.1). Trim `MeshSyncSpec`: replace the embedded `corev1.ConfigMap` `WatchList` with a typed list or a `ConfigMapReference`; make `Version` authoritative for the image tag.
- Add `additionalPrinterColumns` (size, endpoint, ready, age), `shortNames`, `categories: [meshery]`, and CEL `x-kubernetes-validations` for cross‑field invariants (e.g. external access requires a compatible service type).

**Acceptance:** a `v1alpha1` object round‑trips through conversion; `kubectl get brokers` shows endpoint/ready columns; Meshery Server's existing client calls succeed unmodified.

### Phase 3 — NATS broker overhaul & networking reconfiguration (WS‑4, WS‑6) — see §6

- Upgrade NATS to the current line and replace the `connecteverything` reloader; align the StatefulSet/Service with the official NATS chart's structure (headless service for clustering + client service for access; optional JetStream).
- Implement **declarative, reconcilable service networking** (§6): service type, ports, annotations, and external‑access policy become spec fields; the controller **watches the Service** and recomputes the endpoint on change; endpoint derivation becomes deterministic and non‑blocking.
- Slim dependencies (WS‑6): adopt meshkit `v1.x` *or* lift the small endpoint helper into the operator, and prune Helm/GORM/CUE/ORAS from the operator's module graph.

**Acceptance:** changing `spec.service.type` from `LoadBalancer` to `NodePort` (or patching ports/annotations) on a live `Broker` reconciles the Service and updates `status.endpoint` **without** a hot‑loop and **without** manual pod deletion; MeshSync receives a correctly‑schemed `nats://host:port` URL; operator binary size and `go list` time drop materially.

### Phase 4 — Lifecycle testing, packaging & release (WS‑7, WS‑8, WS‑1)

- Build the e2e suite (§7): kind matrix across ≥2 Kubernetes versions, a real NATS, and explicit **networking‑reconfiguration** and **upgrade/conversion** scenarios.
- Regenerate the OLM bundle off the v4 layout and current operator‑sdk; bump the version, add `replaces`/`olm.skipRange` to establish an upgrade graph, broaden install modes, and fix the maintainer metadata. Wire `bundle`/`scorecard` into CI.

**Acceptance:** e2e is green on the kind matrix; `operator-sdk scorecard` passes; bundle validates and declares a coherent upgrade edge from the prior version.

---

## 6. Deep dive: reconfiguring the NATS broker's service networking post‑deployment

This is the central motivating objective. Today it is impossible to express, and impossible to reconcile.

### 6.1 Proposed `BrokerSpec` (v1alpha2)

```go
// BrokerSpec defines the desired state of the NATS broker.
type BrokerSpec struct {
    // Size is the number of NATS server replicas.
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=10
    // +kubebuilder:default=1
    Size int32 `json:"size,omitempty"`

    // Version pins the NATS server image tag (defaults to the operator's bundled version).
    // +optional
    Version string `json:"version,omitempty"`

    // Service controls how the broker is exposed on the network. All fields are
    // reconcilable in place — changing them updates the live Service and re-derives status.endpoint.
    // +optional
    Service BrokerServiceSpec `json:"service,omitempty"`

    // JetStream enables persistent streaming (file or memory store).
    // +optional
    JetStream *JetStreamSpec `json:"jetStream,omitempty"`
}

type BrokerServiceSpec struct {
    // Type is the Kubernetes Service type for client access.
    // +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
    // +kubebuilder:default=ClusterIP
    Type corev1.ServiceType `json:"type,omitempty"`

    // Annotations are merged onto the client Service (e.g. cloud LB controller hints,
    // MetalLB address pools, internal-LB switches).
    // +optional
    Annotations map[string]string `json:"annotations,omitempty"`

    // Ports optionally overrides the default NATS port set (client/monitor/metrics/cluster/...).
    // +optional
    Ports []BrokerPort `json:"ports,omitempty"`

    // LoadBalancerClass / loadBalancerSourceRanges for LoadBalancer type.
    // +optional
    LoadBalancerClass        *string  `json:"loadBalancerClass,omitempty"`
    LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`

    // ExternalEndpointOverride lets the operator/Meshery Server pin the advertised
    // endpoint when auto-derivation is undesirable (air-gapped, ingress/gateway in front).
    // +optional
    ExternalEndpointOverride string `json:"externalEndpointOverride,omitempty"`
}
```

Rationale:

- **`Type` default becomes `ClusterIP`**, not `LoadBalancer`. ClusterIP works on every cluster (kind, minikube, bare‑metal, cloud) and is the correct default for an in‑cluster broker that MeshSync reaches via the internal endpoint. `LoadBalancer`/`NodePort` become explicit opt‑ins.
- **`Annotations`/`LoadBalancerClass`/`SourceRanges`** make the common production needs (internal LB, MetalLB pool, source‑range allow‑lists) expressible without forking the operator.
- **`ExternalEndpointOverride`** gives Meshery Server an escape hatch for topologies where TCP auto‑probing is wrong (ingress/gateway, air‑gapped, NAT).

### 6.2 Controller changes to make networking reconcilable

1. **Watch the Service.** Add `Owns(&corev1.Service{})` and `Owns(&corev1.ConfigMap{})` to the Broker controller so that any change to the broker's Service (type change, port edit, LB IP assignment by the cloud controller) triggers reconciliation and endpoint recomputation. This directly closes finding #15/#24.
2. **Server‑Side Apply (SSA) instead of read‑modify‑DeepEqual.** Replace the `existing.Spec = desired.Spec` + `DeepEqual` pattern with `client.Apply` (SSA) using a stable field manager. SSA lets the API server reconcile only operator‑owned fields and leaves server‑defaulted fields (`clusterIP`, `nodePort`, `healthCheckNodePort`) untouched — eliminating the hot‑loop (#13) while still converging declared fields.
3. **Deterministic, non‑blocking endpoint derivation.** Compute the endpoint directly from the (now‑watched) Service object: for `ClusterIP`, `internal = <clusterIP>:<clientPort>`; for `NodePort`, `external = <nodeIP>:<nodePort>`; for `LoadBalancer`, `external = <ingress IP/host>:<clientPort>` once `status.loadBalancer.ingress` is populated (and requeue while pending). **Remove TCP dialing from the reconcile loop**; if reachability validation is desired, perform it asynchronously and surface it as a `Condition`, never as a blocking gate. Always emit a scheme‑qualified `nats://host:port`.
4. **Propagate to MeshSync correctly.** Write the derived endpoint to `Broker.Status.Endpoint` and ensure the MeshSync reconciler injects a `nats://`‑schemed URL by env *name* (not positional `Env[0]`), and requeues MeshSync when the referenced Broker's endpoint changes (a watch/`EnqueueRequestsFromMapFunc` from Broker→MeshSync).

### 6.3 Why this satisfies "reconfigure post‑deployment"

With (1)+(2), an operator user (or Meshery Server) can `kubectl patch broker … --type=merge -p '{"spec":{"service":{"type":"NodePort"}}}'` on a **running** broker; the Service is reconciled via SSA without churn, the watch fires, the endpoint is recomputed deterministically, `status.endpoint` updates, and MeshSync is re‑enqueued to pick up the new `nats://` URL — no manual Service edits, no pod deletion, no hot‑loop. The same path handles a LoadBalancer IP arriving late (the Service `status` update triggers a reconcile) and annotation/port edits.

---

## 7. Deep dive: testing the operator's lifecycle

The objective is to make the **full lifecycle** — create → reconcile → become healthy → expose endpoint → reconfigure → upgrade → delete/finalize — observable in automated tests.

### 7.1 Three tiers

1. **Unit** (fast, no cluster): table tests for resource builders (`pkg/broker`, `pkg/meshsync`), endpoint derivation (pure function over a `corev1.Service` — extracted from the blocking helper), and SSA patch shaping. The endpoint logic becoming a pure function (§6.2.3) is what makes this tier meaningful.
2. **Integration (`envtest`)**: reconcile against a real apiserver+etcd. Fix asset resolution (#36). Because envtest has no kubelet, **drive workload status explicitly** — have tests set `StatefulSet`/`Deployment` `.status` (replicas/readyReplicas) and Service `.status.loadBalancer.ingress` via the status subresource to exercise health/endpoint/condition transitions deterministically. This is the key technique that makes health and networking testable without a real kubelet (closing #35).
3. **End‑to‑end (kind)**: a real cluster with real pods and a real NATS. Validate that MeshSync connects to the broker and publishes, across a **Service‑type matrix** (ClusterIP, NodePort; LoadBalancer via MetalLB/cloud‑provider‑kind), and across **reconfiguration** and **upgrade/conversion** scenarios.

### 7.2 Scenarios to cover (new)

- **Networking reconfiguration:** create a `ClusterIP` broker, assert internal endpoint + MeshSync connectivity; patch to `NodePort`, assert no Service churn (observe `metadata.generation`/managedFields), endpoint recomputed, MeshSync re‑enqueued and reconnected.
- **Delayed LoadBalancer IP:** create a `LoadBalancer` broker with no ingress IP; assert `Pending` condition + requeue; inject `status.loadBalancer.ingress`; assert endpoint resolves.
- **Upgrade/conversion:** apply a `v1alpha1` `Broker`, assert conversion to `v1alpha2` storage and round‑trip read via the served `v1alpha1`.
- **Finalizer/cleanup:** delete a `Broker`, assert owned objects are removed and the finalizer clears (the recently‑added finalizer path gains regression coverage).
- **Leader election:** two manager replicas elect exactly one leader (regression for #12).

### 7.3 CI wiring

Promote e2e into CI on a kind matrix (≥2 Kubernetes minor versions, pinned), publish JUnit + coverage, and gate merges on the unit+integration tiers (e2e advisory→required as it stabilizes).

---

## 8. Risks, compatibility & sequencing

- **Backward compatibility with Meshery Server.** Meshery Server installs and manages the operator and consumes the `pkg/client` typed clientset and the `v1alpha1` CRDs. The conversion webhook (Phase 2) and retaining a served `v1alpha1` keep that contract intact; the typed client should be regenerated (or thinly re‑pointed) but its method surface preserved. **Coordination with the `meshery/meshery` repo is a hard dependency for the API phase** and should be tracked jointly.
- **Default Service type change (LoadBalancer→ClusterIP).** This is behaviorally safer everywhere but changes the out‑of‑the‑box external‑exposure behavior. It must be called out in release notes; Meshery Server should set `spec.service.type` explicitly to preserve any environment that relied on the LB default.
- **RBAC narrowing.** Moving off the wildcard role risks `forbidden` errors if a resource is missed; mitigate by deriving the role from `+kubebuilder:rbac` markers, validating in e2e under the narrowed role, and shipping the change behind Phase 0 soak testing before it reaches releases.
- **meshkit v1.x adoption.** A major‑version bump may carry breaking changes; isolate by either upgrading meshkit wholesale (preferred, aligns with MeshSync) or extracting just the endpoint helper into the operator to decouple the broker‑networking work from the meshkit upgrade timeline.
- **Sequencing guardrail.** Phase 0 (safety) and Phase 1 (tooling) carry no CRD contract change and can ship independently and quickly. API/networking/testing build on them. Each phase is independently releasable.

---

## 9. Appendix A — Version targets

| Component | Current (repo) | Target |
|-----------|----------------|--------|
| Project layout | `go.kubebuilder.io/v2` (`PROJECT`) | `go.kubebuilder.io/v4` |
| operator‑sdk (bundle stamp) | `v1.14.0` (2021) | current `v1.4x` line |
| OLM bundle version | `0.0.1` (frozen, 2022‑05‑12) | semver‑bumped with `replaces`/`skipRange` |
| NATS server | `nats:2.8.2-alpine3.15` | current 2.10.x/2.11.x line |
| NATS config‑reloader | `connecteverything/…:0.6.0` (defunct org) | `natsio/nats-server-config-reloader` current |
| Go | `1.26.4` | keep current (already modern) |
| k8s libraries | `v0.35.0` | keep current; align across repos |
| controller‑runtime | `v0.22.4` | keep current |
| meshkit | `v0.8.64` | `v1.0.x` (or extract endpoint helper) |
| kustomize (Makefile) | `v3.8.7` | v5.x |
| kind (Makefile) | `v0.18.0` | current |
| cert‑manager API | `v1alpha2` | `v1` |

(Go, the k8s libraries, and controller‑runtime are already on current lines — the modernization debt is in *layout, packaging, NATS, meshkit, and tooling pins*, not the core runtime deps.)

## 10. Appendix B — Consolidated punch list by file

- `PROJECT` — migrate to v4 layout; register Broker + MeshSync with controllers.
- `main.go` — stable `LeaderElectionID`; add healthz/readyz; drop `auth/gcp`; secure metrics.
- `api/v1alpha1/*` — add `v1alpha2`; networking‑capable `BrokerSpec`; trim `MeshSyncSpec`; printer columns/CEL.
- `controllers/broker_controller.go` — `Owns(Service, ConfigMap)`; SSA; non‑blocking endpoint; deterministic ordering.
- `controllers/meshsync_controller.go` — env‑by‑name; `nats://` scheme; Broker→MeshSync watch.
- `pkg/broker/resources.go` — NATS image/reloader upgrade; parameterized Service (type/ports/annotations); remove committed JWT/duplicate resolver.
- `pkg/broker/broker.go` — pure, non‑blocking `GetEndpoint`; health via `ReadyReplicas`.
- `pkg/client/v1alpha1/*` — regenerate via code‑generator or replace; remove mutable global codec.
- `config/rbac/*` — delete wildcard `controller-role`; complete `+kubebuilder:rbac` markers; drop kube‑rbac‑proxy scaffolding.
- `config/manager/manager.yaml` — remove `hostPort`; add securityContext + probes; realistic memory; pin image.
- `config/certmanager/*`, `config/default/kustomization.yaml` — cert‑manager `v1`; enable webhook/cert/prometheus wiring.
- `Makefile` — remove `trivialVersions`; bump tool pins; align envtest k8s version; add lint install; fix/add `bundle.Dockerfile`.
- `controllers/suit_test.go` — `setup-envtest` assets; fix unchecked error + mislabeled logger; status‑driven health tests.
- `integration-tests/main.sh` — pin kind k8s version; add reconfiguration + upgrade scenarios.
- `.github/workflows/*` — pin actions; standardize runners; add CodeQL/Trivy/SBOM/cosign + drift gate.
- `.golangci.yml` — enable `gosec`.
- `bundle/**` — regenerate off v4/current SDK; bump version; upgrade graph; fix maintainer metadata.

---

*This document is a proposal intended to seed issues/epics per workstream. Each workstream in §5 is sized to become a tracking issue with the findings in §4 as its checklist.*
