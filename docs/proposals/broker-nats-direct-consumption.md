# Broker → Direct Consumption of Upstream NATS-on-Kubernetes

**Status:** Proposed
**Target repository:** `meshery/meshery-operator`
**Related repositories:** `meshery/meshery` (Meshery Server — installs/manages the operator and consumes the typed client + `Broker.Status.Endpoint`), `meshery/meshsync` (the workload pointed at the broker), `meshery/meshkit`.
**Companion document:** [`operator-modernization-plan.md`](./operator-modernization-plan.md) — this proposal is a focused deep‑dive on its **WS‑4 (NATS)**, with knock‑on effects on **WS‑2 (API/CRD)** and **WS‑6 (deps)**.

---

## 1. Executive summary

The maintainer's hypothesis — *"Meshery's Broker CRD is nothing more than a shim over the NATS CRD / NATS custom controller; we should directly consume what the NATS project officially offers"* — is **directionally correct, with two precise corrections**:

1. **The Broker CRD does not currently sit "over" any NATS CRD or NATS controller.** It is a *from‑scratch reimplementation* of the NATS **server** topology — a hand‑authored `StatefulSet`, `LoadBalancer` `Service`, `nats.conf` + accounts `ConfigMap`s (with a **committed account JWT** and `resolver: MEMORY`), and a config‑reloader sidecar (`pkg/broker/resources.go`). Its only NATS‑specific value‑add beyond "render a NATS server" is endpoint derivation into `Broker.Status.Endpoint`, which Meshery Server and MeshSync consume. So it is best described as **a bespoke clone of the official NATS Helm chart's server manifests**, not a shim over a controller.

2. **There is no supported NATS CRD/controller that provisions a NATS *server* today.** The thing the phrase "NATS CRD / NATS custom controller" most likely refers to — `nats-io/nats-operator` and its `NatsCluster` CRD — is **archived and deprecated**, explicitly steering users to the Helm chart. The supported upstream pattern in 2026 is a **two‑part split**: the **NATS Helm chart** provisions the *server* (StatefulSet/Services/config/reloader/JetStream PVCs/monitoring), and **NACK (NATS Controllers for Kubernetes)** manages **JetStream objects** (Stream, Consumer, Account, KeyValue, ObjectStore) via CRDs against an *already‑running* server. NACK does **not** provision the server. ([nats-operator deprecation](https://github.com/nats-io/nats-operator), [NACK](https://github.com/nats-io/nack), [NATS Helm chart](https://github.com/nats-io/k8s/tree/main/helm/charts/nats))

The correct modernization, therefore, is **not** "consume a NATS provisioning CRD" (none exists) but **"stop hand‑maintaining NATS server YAML and instead track the official NATS Helm chart's rendered server manifests,"** while retaining a thin `Broker` CR as the Meshery‑facing API and the source of `Status.Endpoint` (a hard backward‑compatibility constraint with Meshery Server). JetStream object management — if/when MeshSync needs it — is layered on later via NACK.

A critical constraint shapes the *mechanism*: the operator **just removed the Helm SDK from its dependency graph** for binary‑size reasons (confirmed — `go.mod`/`go.sum` carry no Helm, GORM, CUE, or ORAS). Re‑adding the Helm Go SDK at runtime would reverse WS‑6 and is rejected. The recommended path renders the chart with the **Helm CLI at build time** (`helm template`), checks the output into the repo, embeds it via `go:embed`, and applies it with the **Server‑Side Apply** machinery the controllers already use — **zero new runtime dependencies**.

**Recommendation: Option D** — vendor the official NATS chart's rendered server manifests (regenerated from upstream at build time), apply them via SSA behind a thin `Broker` CR that preserves `Status.Endpoint` and the typed client, eliminate the committed JWT, and add JetStream via chart values; adopt NACK later only if JetStream *objects* are required.

---

## 2. Current‑state shim analysis

### 2.1 What the Broker CR actually produces

`pkg/broker/resources.go` and `pkg/broker/broker.go` build four objects per `Broker` (in deterministic order — `pkg/broker/broker.go:32`):

| Object | Source | Notes |
|--------|--------|-------|
| `ConfigMap meshery-nats-config` (`nats.conf`) | `resources.go:69` | monitoring on `:8222`, `server_name: $POD_NAME`, `resolver: MEMORY`, includes `accounts/resolver.conf` |
| `ConfigMap meshery-nats-accounts` (`resolver.conf`) | `resources.go:88` | **committed account JWT** preloaded into a `resolver: MEMORY` block (`resources.go:98`) |
| `Service meshery-nats` | `resources.go:103` | **`Type: LoadBalancer`** default, 6 fixed ports (client 4222, cluster 6222, monitor 8222, metrics 7777, leafnodes 7422, gateways 7522) |
| `StatefulSet meshery-nats` | `resources.go:142` | `nats:2.10.29-alpine3.21` + `natsio/nats-server-config-reloader:0.23.0` sidecar; liveness/readiness on `:8222`; LDM pre‑stop hook |

This is, line‑for‑line, the shape the official NATS Helm chart emits — minus JetStream, minus TLS, minus modern auth, and minus the chart's maintenance cadence. The hand‑authored version pins NATS server **2.10.x**; upstream chart and server now ship **in lock‑step at 2.14.2**, four minor versions ahead. The committed JWT and `resolver: MEMORY` are a security liability that the chart's auth model removes.

### 2.2 The genuinely operator‑specific surface (must be preserved)

The Broker controller is **not** pure NATS plumbing — it owns three contracts that the rest of Meshery depends on:

1. **`Broker.Status.Endpoint.{Internal,External}`** — derived purely from the client `Service` by `pkg/broker/endpoint.go` (`DeriveEndpoint`), now non‑blocking and Service‑type aware (ClusterIP → clusterIP:port; NodePort → apiHost:nodePort; LoadBalancer → ingress:port, with a `pending` signal). `broker.go:117` writes it to status; `ExternalEndpointOverride` wins when set.
2. **The typed clientset** `pkg/client/v1alpha1` — `BrokerInterface`/`MeshSyncInterface` with `Create/Update/Delete/Get/List/Watch/Patch` (`pkg/client/v1alpha1/broker.go:21`). **Meshery Server imports this** to create/manage Broker & MeshSync CRs and to read `Status.Endpoint`.
3. **MeshSync wiring** — `controllers/meshsync_controller.go:316` (`reconcileBrokerConfig`) copies `Broker.Status.Endpoint.Internal` into `MeshSync.Status.PublishingTo`; `pkg/meshsync/meshsync.go:53` injects it into the MeshSync Deployment as `BROKER_URL` **by env name**, normalized to a `nats://` scheme (`meshsync.go:83`). A Broker→MeshSync watch (`meshsync_controller.go:112`) re‑enqueues MeshSync when the endpoint changes.

### 2.3 What has already landed (the doc lags the code)

The companion modernization plan describes a `BrokerSpec` with only `Size`. **That is stale.** The repo has already shipped (commits `b1e3164`, `6c78cde`):

- A **networking‑capable `BrokerSpec`** — `Version`, `Service{Type, Annotations, LoadBalancerClass, LoadBalancerSourceRanges, ExternalEndpointOverride}`, `Size`, with CEL validations (`api/v1alpha1/broker_types.go`, `config/crd/bases/meshery.io_brokers.yaml`).
- **SSA reconcile** with a stable field manager (`broker_controller.go:315`), `Owns(Service, ConfigMap, StatefulSet)`, and the Broker→MeshSync watch.
- **Pure, non‑blocking endpoint derivation** (`endpoint.go`) — the meshkit blocking‑TCP path is gone.
- **Helm/GORM/CUE/ORAS removed from `go.mod`** (WS‑6 dependency slimming effectively done).

**Implication:** the *controller plumbing* for in‑place networking reconfiguration is complete. The remaining WS‑4 gap is squarely **what gets applied** — the hand‑authored NATS server manifests — which is exactly what this proposal replaces.

---

## 3. The upstream NATS‑on‑Kubernetes landscape (2026)

### 3.1 NATS Helm chart — provisions the **server**

- **Repo/registry:** `nats-io/k8s`, chart at `helm/charts/nats`; installed from the classic repo `https://nats-io.github.io/k8s/helm/charts/` (`helm repo add nats …`) and mirrored on ArtifactHub as `nats/nats`. The chart now versions **in lock‑step with the server**: current `version`/`appVersion` are both **2.14.2**. docs.nats.io states *"The recommended way to deploy NATS on Kubernetes is using Helm with the official NATS Helm Chart."* ([k8s repo](https://github.com/nats-io/k8s), [chart Chart.yaml](https://raw.githubusercontent.com/nats-io/k8s/main/helm/charts/nats/Chart.yaml), [ArtifactHub nats/nats](https://artifacthub.io/packages/helm/nats/nats), [docs.nats.io: NATS and Kubernetes](https://docs.nats.io/running-a-nats-service/nats-kubernetes))
- **What it deploys:** `StatefulSet`; **headless** `Service` (`<release>-headless`, for cluster route discovery) + **client** `Service` (`<release>`, ClusterIP by default, overridable to NodePort/LoadBalancer); the **`reloader`** sidecar (config hot‑reload); optional **JetStream** file‑store PVCs; optional **Prometheus** exporter sidecar + PodMonitor; an optional **nats‑box** admin Deployment. ([chart README](https://github.com/nats-io/k8s/tree/main/helm/charts/nats))
- **Default ports:** client **4222**, monitoring **8222**, cluster **6222** (plus optional websocket/mqtt/leafnodes/gateways). ([chart README](https://github.com/nats-io/k8s/tree/main/helm/charts/nats))
- **Key values:** `config.jetstream.enabled` + `config.jetstream.fileStore.pvc.size` (and memStore); `config.cluster.enabled`/`config.cluster.replicas`; `config.merge.authorization.token` / nkey / `config.resolver` (operator‑JWT mode); per‑listener `*.tls`; `service` (type/ports/annotations); `container.image.tag`; `podTemplate`/`statefulSet` merge‑patch escape hatches. ([chart README](https://github.com/nats-io/k8s/tree/main/helm/charts/nats))
- The chart's **1.0.0 rebuild** consolidated the older split charts and is the canonical, maintained way to run NATS on Kubernetes. ([NATS blog: Helm chart 1.0.0](https://nats.io/blog/nats-helm-chart-1.0.0-rc/))

### 3.2 NACK — manages **JetStream objects**, not the server

- **Repo:** `nats-io/nack` (the `jetstream-controller`). CRDs under **`jetstream.nats.io`** (`v1beta2` storage): **Stream, Consumer, StreamTemplate, Account, KeyValue, ObjectStore**. ([NACK](https://github.com/nats-io/nack), [NACK CRDs](https://raw.githubusercontent.com/nats-io/nack/main/deploy/crds.yml))
- **Does NOT provision the server.** It requires an existing NATS server with JetStream enabled and connects to it via `jetstream.nats.url` (`Account` is a *connection/auth config*, not a server‑side account provisioner). Installed via Helm `nats/nack`. ([NACK](https://github.com/nats-io/nack))
- **Mode caveat:** **KeyValue, ObjectStore, and Account are only reconciled in `--control-loop` mode**; the default (legacy) controller handles only Stream/Consumer/StreamTemplate. ([docs.nats.io: NACK](https://docs.nats.io/running-a-nats-service/configuration/resource_management/configuration_mgmt/kubernetes_controller))
- **Division of labor:** *Helm chart deploys the server; NACK declaratively manages JetStream resources on it.*

### 3.3 nats-operator — deprecated/archived

- `nats-io/nats-operator` (the **`NatsCluster`** CRD that "models a NATS cluster") is **archived (read‑only since 2025‑04‑10) and deprecated**. Its banner states: *"The recommended way of running NATS on Kubernetes is by using the Helm charts. If looking for JetStream support, this is supported in the Helm charts. The NATS Operator is not recommended to be used for new deployments."* ([nats-operator](https://github.com/nats-io/nats-operator))
- This is almost certainly the "NATS CRD / NATS custom controller" the hypothesis refers to. **It is a dead end** — do not build on it.

### 3.4 Conclusion: is there a supported server‑provisioning CRD?

**No.** As of 2026 the supported pattern is strictly **Helm‑chart‑for‑server + NACK‑for‑JetStream‑objects**. There is no first‑class, currently‑maintained `NatsCluster`‑style CRD. "Directly consume what NATS officially offers" therefore means **consume the chart's *output*** (and optionally NACK's CRDs), not a provisioning CRD.

### 3.5 Applying a Helm chart from a Go operator **without** the Helm SDK

| Mechanism | Runtime deps | Upgrade story | Verdict |
|-----------|--------------|---------------|---------|
| **`helm template` at build time → vendor rendered YAML → `go:embed` → SSA** | **none** (Helm CLI is a build tool only) | bump pinned chart version, re‑render, CI drift gate | **Recommended** — preserves WS‑6 slimming |
| **Embed the chart `fs.FS` + render with `slok/go-helm-template`** | **tiny** (a small templating lib, **no Helm binary/SDK**) | re‑template per‑CR at runtime; bump embedded chart | Viable if per‑CR templating is wanted (chart must have no subchart deps/hooks) |
| Embed chart + render via **Helm Go SDK** (`helm.sh/helm/v3`) at runtime | **heavy** (re‑adds Helm, reverses WS‑6) | native Helm install/upgrade/rollback + release storage | Rejected |
| **Flux `HelmRelease`** (helm‑controller uses the real Helm SDK) **/ Argo CD `Application`** (inflates with `helm template`) | requires a GitOps controller in every Meshery install | excellent drift/upgrade, but operationally invasive | Rejected (couples Meshery to Flux/Argo) |
| `helm template` + **kustomize** (`--enable-helm`) base | none at runtime, **but kustomize shells out to the Helm binary** at build | similar to row 1, more moving parts | Viable alternative to row 1 |

GitOps tooling validates the row‑1 approach: Argo CD "is only used to inflate charts with `helm template`" and then applies the output with its own sync engine; SSA is the field‑ownership model that makes this converge cleanly. The operator already does SSA with a stable field manager, so row 1 drops in. `slok/go-helm-template` (row 2) is the no‑binary middle ground — it renders an embedded chart "without needing a Helm binary or external command execution," with the explicit caveat that it supports only simple `helm template` (no subchart dependencies/hooks). ([helm template](https://helm.sh/docs/helm/helm_template/), [slok/go-helm-template](https://github.com/slok/go-helm-template), [Argo CD — Helm](https://argo-cd.readthedocs.io/en/latest/user-guide/helm/), [Flux helm-controller](https://fluxcd.io/flux/components/helm/helmreleases/))

---

## 4. Options

Effort: **S** ≈ days, **M** ≈ 1–2 weeks, **L** ≈ 3–6 weeks + cross‑repo coordination.

### Option A — Operator renders & applies the official chart; thin Broker CR maps to values
Replace `pkg/broker/resources.go` with chart‑sourced manifests; `BrokerSpec` fields map to chart values. The apply mechanism is the open question (SDK vs. build‑time render). If implemented with the build‑time `helm template` + `go:embed` + SSA mechanism, **A collapses into D**. If implemented with the runtime Helm SDK, it reverses WS‑6.
- **Pros:** tracks upstream; Broker CR (and `Status.Endpoint`) preserved.
- **Cons:** "render at runtime" temptation re‑introduces the Helm SDK; full value→spec mapping is large surface.
- **Effort:** M–L · **Risk:** Med.

### Option B — NACK + Helm‑deployed server; operator points MeshSync at it
Server installed via the Helm chart (by Meshery Server or a bundled manifest set); JetStream objects via NACK CRDs; the operator orchestrates and points MeshSync at the service.
- **Pros:** fully upstream‑supported split; first‑class JetStream object management.
- **Cons:** server install moves **out of the operator** (Meshery Server must own it → cross‑repo work); adds NACK as a dependency/bundle; the `Broker` CR's reason‑to‑exist shrinks.
- **Effort:** L · **Risk:** Med‑High. **Best as a *follow‑on* for JetStream, not the base.**

### Option C — Deprecate the Broker CRD entirely
Meshery Server installs upstream NATS (chart/bundled manifests) directly and points MeshSync at the known service DNS (`nats://<release>.<ns>.svc:4222`); the operator drops broker provisioning.
- **Pros:** least code in the operator; maximal alignment with upstream.
- **Cons:** **breaks the typed‑client + `Status.Endpoint` contract**; large, lock‑step change in `meshery/meshery`; loses the in‑place networking‑reconfig UX just built; endpoint derivation (LB/NodePort/override logic) must move into Meshery Server.
- **Effort:** L (+ heavy `meshery/meshery` coordination) · **Risk:** High. **Viable only as a long‑term north star.**

### Option D — Thin Broker CR wrapping **vendored chart manifests** applied via SSA (no Helm runtime) — **RECOMMENDED**
Render the official chart at build time, check the manifests in, `go:embed` them, decode into objects, overlay `BrokerSpec`, and SSA‑apply — preserving `Status.Endpoint`, the typed client, and the in‑place networking reconfiguration.
- **Pros:** "directly consumes what NATS officially offers" (chart output is the source of truth, refreshed from upstream); **no new runtime deps** (honors WS‑6); Meshery Server compatibility fully preserved; eliminates committed JWT; JetStream via chart values; reuses the existing SSA/watch/endpoint machinery.
- **Cons:** a build‑time `helm template` step + CI drift gate to maintain; value→object overlay is hand‑written Go (bounded, since only a few fields are reconfigurable).
- **Effort:** M · **Risk:** Low‑Med.

### Options matrix

| | Server provisioning | JetStream | Broker CR | Helm **runtime** dep | Meshery Server impact | Effort | Risk |
|---|---|---|---|---|---|---|---|
| A | chart‑rendered (operator) | chart values | thin | depends on mechanism | low | M–L | Med |
| B | Helm (out‑of‑band) | **NACK CRDs** | shrinks | none | medium | L | Med‑High |
| C | Meshery Server (Helm) | chart/NACK | **removed** | none | **high** | L | High |
| **D** | **embedded chart output** | chart values (+NACK later) | **thin, preserved** | **none** | **low** | **M** | **Low‑Med** |

---

## 5. Recommendation

**Adopt Option D now; keep B's NACK piece as an optional follow‑on for JetStream *objects*.**

Rationale: D is the only option that simultaneously (a) satisfies "consume what NATS officially offers" by making the **official chart's rendered server manifests the source of truth** (regenerated from upstream, not hand‑maintained), (b) **preserves every Meshery Server contract** (typed client, `brokers.meshery.io` CRD, `Status.Endpoint`), (c) **respects the just‑completed Helm‑SDK removal** (build‑time `helm template`, zero runtime deps), and (d) **retains the in‑place networking reconfiguration and endpoint UX** that already shipped. Options A‑runtime and C either reverse WS‑6 or break the Meshery Server contract; B is the right *destination for JetStream objects* but the wrong *base* because it relocates server provisioning out of the operator.

---

## 6. File‑level design (Option D)

### 6.1 Render & embed the chart (build‑time, no runtime dep)
- **New `Makefile` target `nats-manifests`:** `helm template meshery-nats nats/nats --version <PINNED> -f pkg/broker/chart/values.yaml > pkg/broker/manifests/nats.gen.yaml` (Helm CLI is a build/dev tool, never a `go.mod` entry). Pin the chart repo+version; document the refresh procedure.
- **New `pkg/broker/chart/values.yaml`:** Meshery defaults that reproduce today's topology for a clean cutover — client `Service` named `meshery-nats`, ports {client 4222, monitor 8222, metrics 7777, cluster 6222, leafnodes 7422, gateways 7522}, reloader on, monitoring on, resource requests/limits, **auth = token/no‑auth (no committed JWT)**, JetStream off by default.
- **New `pkg/broker/manifests/nats.gen.yaml`** (generated, committed) + **`pkg/broker/embed.go`:** `//go:embed manifests/*.yaml`, decoded with `k8s.io/apimachinery/.../yaml` + the scheme into `[]Object`.

### 6.2 Replace hand‑authored resources
- **`pkg/broker/resources.go`:** delete the hand‑authored `StatefulSet`/`Service`/`NatsConfigMap`/**`AccountsConfigMap` (committed JWT)** vars. Keep only label/annotation/name constants still referenced.
- **`pkg/broker/broker.go` `GetObjects`:** return the decoded embedded objects, then call a small **overlay** that maps `BrokerSpec` onto them:
  - `Size` → StatefulSet `replicas`; `Version` → server container image tag.
  - `Service.Type/Annotations/LoadBalancerClass/LoadBalancerSourceRanges` → the **client** Service (reuse existing `applyServiceSpec`).
  - Because manifests are static, parameterization is a **Go object‑mutation overlay**, not a re‑run of Helm. The set of mutable fields is small and already defined by `BrokerSpec`. (If per‑CR re‑templating ever becomes necessary — e.g. value‑driven JetStream variants — swap the static `go:embed` of rendered YAML for an embedded chart `fs.FS` rendered by `slok/go-helm-template`, still with **no Helm binary or SDK**.)

### 6.3 Endpoint, controller, MeshSync — mostly unchanged
- **`pkg/broker/endpoint.go`:** already pure/Service‑driven. Ensure it reads the **client** Service by the chart's client‑service name (keep `meshery-nats` to avoid churn). No logic change.
- **`controllers/broker_controller.go`:** already SSA + `Owns(Service/ConfigMap/StatefulSet)` + sets controller refs in the apply loop. Add `Owns(...)` for any *additional* chart‑owned kinds actually rendered (e.g. `ServiceAccount`, `PodDisruptionBudget`, `PersistentVolumeClaim` when JetStream is on) so they re‑enqueue and GC correctly.
- **MeshSync wiring:** unchanged — `Status.Endpoint.Internal` → `PublishingTo` → `BROKER_URL` (`nats://`). The chart's client Service is the same DNS, so MeshSync connectivity is preserved.

### 6.4 JetStream (additive API)
- **`api/v1alpha1/broker_types.go`:** add `JetStream *JetStreamSpec` (`Enabled bool`, `Store {file|memory}`, `Size resource.Quantity`). Additive → backward compatible. The overlay flips `config.jetstream.*` (or, since manifests are pre‑rendered, swaps in a JetStream‑enabled variant + PVC). Regenerate CRD/printer columns.

### 6.5 Security
- **Remove the committed JWT and `resolver: MEMORY`.** Default in‑cluster auth to **token from a Secret** (or no‑auth for dev), exposed via a future `BrokerSpec.Auth`. **TLS** via the chart's `*.tls` values wired to a cert‑manager `Certificate` (enable the dormant `config/certmanager` wiring with the `cert-manager.io/v1` API). Operator‑JWT mode becomes available later through NACK `Account`.

### 6.6 RBAC / OLM
- Add `+kubebuilder:rbac` markers for every kind the chart renders that the operator must apply/own: `serviceaccounts`, `poddisruptionbudgets` (policy), `persistentvolumeclaims` + `secrets` (JetStream/auth), beyond today's `statefulsets`/`services`/`configmaps`. Regenerate `config/rbac/role.yaml`; update the CSV's `permissions`/`clusterPermissions`. The embedded manifests ride **inside the operator image** — **no new bundle objects, no Helm in the bundle**.

---

## 7. Backward compatibility with Meshery Server

**The contract Meshery Server depends on (hard constraint):**
1. The `brokers.meshery.io` / `meshsyncs.meshery.io` **v1alpha1 CRDs** keep existing (group `meshery.io`, resources `brokers`/`meshsyncs`, namespaced) — consumed via the typed client's REST paths (`pkg/client/v1alpha1/broker.go:48`).
2. **`Broker.Status.Endpoint.{Internal,External}`** keeps populating.
3. The **Go types** in `api/v1alpha1` keep their importable shape (Meshery Server imports them transitively through the typed client).

**Under Option D, all three are preserved unchanged** — D only changes *which manifests the controller applies inside the cluster*, which is invisible to Meshery Server. **No coordination in `meshery/meshery` is required for the D cutover.**

Coordination *is* required for two opt‑in changes, which should be gated behind release notes:
- **Default Service type.** Today's default is `LoadBalancer` (`resources.go:138`); keep that default in the vendored values for the cutover to avoid behavioral change. If/when the default flips to `ClusterIP` (safer everywhere), Meshery Server should set `spec.service.type` explicitly for environments that relied on the LB default.
- **Auth change.** Removing the committed JWT changes the broker's effective auth posture; confirm no Meshery Server / MeshSync code assumes that specific account.

**Options B/C require real `meshery/meshery` work** (relocating server install, or pointing MeshSync at a DNS name and dropping the typed‑client dependency) — a deprecation/coexistence window and joint tracking issue would be mandatory there. D needs none of that, which is a primary reason it is recommended.

---

## 8. Cross‑cutting requirements

- **MeshSync wiring** (the `nats://host:port` `BROKER_URL`): A/B/D keep the `Status.Endpoint.Internal` → `PublishingTo` → `BROKER_URL` path (`meshsync.go:53`). C replaces it with a hardcoded `nats://<release>.<ns>.svc.cluster.local:4222` injected by Meshery Server.
- **Networking reconfiguration (modernization §6):** the chart's **client `Service`** is the single object whose `type`/`ports`/`annotations` the overlay mutates; SSA applies the change, the existing `Owns(Service)` watch fires, `DeriveEndpoint` recomputes from the Service, and the Broker→MeshSync watch propagates the new URL — **no pod restart, no hot‑loop**. This already works; D just feeds it chart‑sourced manifests.
- **JetStream:** absent today; enabled via `config.jetstream.enabled` + a file‑store PVC (§6.4). JetStream *object* lifecycle (streams/KV) is deferred to NACK (Option B follow‑on) only if MeshSync needs declarative streams.
- **Security:** committed JWT and `resolver: MEMORY` eliminated; token/nkey/operator‑JWT auth + TLS sourced from chart values + cert‑manager (§6.5).
- **Packaging/OLM & RBAC:** embedded manifests in the image; RBAC widened to the chart's object kinds; CSV updated; no Helm in the bundle (§6.6).

---

## 9. Testing strategy (Option D)

- **Unit:** decode the embedded `nats.gen.yaml`; table‑test the `BrokerSpec`→object overlay (size/version/service‑type/JetStream); `DeriveEndpoint` across ClusterIP/NodePort/LoadBalancer/pending/override (pure function — already trivially testable).
- **Integration (`envtest`):** apply the embedded objects via SSA; drive `StatefulSet.status.readyReplicas` and `Service.status.loadBalancer.ingress` through the status subresource (envtest has no kubelet) to exercise health → endpoint → condition transitions and the in‑place service‑type reconfiguration deterministically.
- **e2e (kind):** real chart‑rendered NATS; assert MeshSync connects and publishes; **Service‑type matrix** (ClusterIP, NodePort, LoadBalancer via cloud‑provider‑kind/MetalLB); **chart‑version‑bump regression** (re‑render, diff, redeploy); **JetStream** smoke (PVC bound, stream creatable) once enabled; finalizer/cleanup and Broker→MeshSync propagation.
- **Drift gate (CI):** `make nats-manifests && git diff --exit-code` ensures the committed manifests match the pinned chart render.

---

## 10. Phased migration roadmap

**Phase A — Parity cutover (no API change).** Add `make nats-manifests`, `chart/values.yaml` (reproducing today's Service name/ports/LoadBalancer default), `manifests/nats.gen.yaml`, `embed.go`; switch `GetObjects` to embedded+overlay; **delete the committed JWT**; expand RBAC.
*Acceptance:* existing `integration-tests/main.sh` passes; `Broker.Status.Endpoint` byte‑identical to pre‑change for the same spec; Meshery Server unaffected; drift gate green. *Rollback:* redeploy the prior operator image (change is image‑internal; CRD contract unchanged).

**Phase B — Security + JetStream.** Token/nkey auth from a Secret; TLS via cert‑manager; `BrokerSpec.JetStream` + file‑store PVC.
*Acceptance:* broker starts with no committed credentials; TLS handshake verified in e2e; JetStream PVC bound and a stream creatable.

**Phase C — NACK for JetStream objects (optional, only if needed).** Bundle/depend on NACK; manage MeshSync's streams/KV via `jetstream.nats.io` CRDs.
*Acceptance:* a declared `Stream`/`KeyValue` reconciles against the broker; MeshSync uses it.

**Phase D — Long‑term north star (optional).** Evaluate relocating server provisioning into Meshery Server (Option C) and deprecating bespoke provisioning — **only with funded `meshery/meshery` coordination** and a deprecation window for the typed client.

---

## 11. Effort, sequencing & impact on modernization workstreams

- **WS‑4 (NATS):** Phases A–B **are** the remaining WS‑4 deliverable — they replace the stale hand‑authored server, modernize NATS, add JetStream, and keep the networking reconfiguration. Effort **M**.
- **WS‑2 (API/CRD):** only additive (`JetStream`, later `Auth`) — **S**. No new served version needed; v1alpha1 stays the storage/served version, so the typed client is untouched.
- **WS‑6 (deps):** **constraint, not new work** — do **not** re‑add the Helm SDK; `helm template` stays a build‑time tool. Phase A must not regress `go.mod`.
- Independent of WS‑1/3/5/7/8 already landed; Phase A is releasable on its own and low‑risk.

---

## 12. Open questions for the maintainer

1. **Default Service type:** does Meshery Server set `spec.service.type` explicitly, or rely on today's `LoadBalancer` default? (Determines whether/when we can flip the default to `ClusterIP`.)
2. **Is JetStream actually required by MeshSync today**, or is core NATS pub/sub sufficient? (Decides whether NACK enters the picture at all — Phase C.)
3. **Build‑time Helm CLI + CI drift gate acceptable?** (It is *not* a runtime/`go.mod` dependency, so it should be — confirm.)
4. **Target auth model** for the in‑cluster broker once the committed JWT is removed: no‑auth (current effective posture), token Secret, nkey, or full operator‑JWT via NACK `Account`?
5. **Long‑term ownership:** is the `Broker` CRD a *permanent* part of the Meshery API, or is eventual Option C (Meshery‑Server‑driven Helm install, deprecate the CRD) on the table?
6. **Chart pin & cadence:** pin to the current `nats` chart **2.14.2** (chart and NATS server version in lock‑step) and refresh on what cadence? Who owns the bump + drift‑gate review?
7. **Service naming:** keep `meshery-nats` (likely hardcoded somewhere in Meshery Server/MeshSync) or adopt chart‑default `<release>` naming with an alias?

---

## 13. References

- nats-operator (archived 2025‑04‑10, deprecated; "not recommended for new deployments"; steers to Helm) — <https://github.com/nats-io/nats-operator>
- NACK — NATS Controllers for Kubernetes (`jetstream.nats.io`: Stream, Consumer, StreamTemplate, Account, KeyValue, ObjectStore; does not provision the server; KV/ObjectStore/Account need control‑loop mode; `nats/nack` chart) — <https://github.com/nats-io/nack>
- NACK CRDs (`deploy/crds.yml`) — <https://raw.githubusercontent.com/nats-io/nack/main/deploy/crds.yml>
- docs.nats.io — Kubernetes Controller (NACK) — <https://docs.nats.io/running-a-nats-service/configuration/resource_management/configuration_mgmt/kubernetes_controller>
- NATS Helm chart (server StatefulSet/headless+client Services/reloader/JetStream PVCs/monitoring/nats-box; values) — <https://github.com/nats-io/k8s/tree/main/helm/charts/nats>
- NATS Helm chart `Chart.yaml` (version/appVersion 2.14.2) — <https://raw.githubusercontent.com/nats-io/k8s/main/helm/charts/nats/Chart.yaml>
- NATS on Kubernetes (repo) — <https://github.com/nats-io/k8s>
- NATS Helm charts index — <https://nats-io.github.io/k8s/>
- ArtifactHub `nats/nats` (chart 2.14.x) — <https://artifacthub.io/packages/helm/nats/nats>
- docs.nats.io — NATS and Kubernetes ("recommended way … is using Helm") — <https://docs.nats.io/running-a-nats-service/nats-kubernetes>
- NATS blog — Helm chart 1.0.0 (consolidated, canonical chart) — <https://nats.io/blog/nats-helm-chart-1.0.0-rc/>
- `helm template` (render manifests without a release/cluster) — <https://helm.sh/docs/helm/helm_template/>
- Helm advanced topics (post‑render) — <https://helm.sh/docs/topics/advanced/>
- `slok/go-helm-template` (render a chart in Go without a Helm binary/SDK; simple‑template only) — <https://github.com/slok/go-helm-template>
- Argo CD — Helm (inflates with `helm template`) — <https://argo-cd.readthedocs.io/en/latest/user-guide/helm/>
- Flux — Helm Releases / helm-controller (uses the real Helm SDK) — <https://fluxcd.io/flux/components/helm/helmreleases/>
- Operator SDK — Helm Operator tutorial — <https://sdk.operatorframework.io/docs/building-operators/helm/tutorial/>
- Internal: [`operator-modernization-plan.md`](./operator-modernization-plan.md) §3, §4.4, §6.

---

*This proposal is intended to seed implementation issues. Phase A is the minimum viable change: vendor the chart's rendered server manifests, embed and SSA‑apply them behind the existing Broker CR, and delete the committed JWT — with the typed‑client and `Status.Endpoint` contracts fully preserved.*
