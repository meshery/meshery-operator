# Testing

The operator is tested in three tiers. The unit and envtest tiers run in CI on
every PR and gate merges; the kind e2e tier validates the full lifecycle.

## 1. Unit (fast, no cluster)

Table tests for the resource builders (`pkg/broker`, `pkg/meshsync`) and pure
helpers. No control plane required.

```bash
go test ./pkg/...
```

## 2. Integration (`envtest`)

Ginkgo/Gomega suites in `controllers/` and `pkg/broker/` reconcile against a real
`kube-apiserver` + `etcd` started by [`envtest`](https://book.kubebuilder.io/reference/envtest.html).
There is **no kubelet**, so Pods never actually run - tests that need health or
endpoint behavior drive workload `.status` explicitly via the status subresource.

```bash
make test    # resolves KUBEBUILDER_ASSETS via setup-envtest, then runs unit + envtest
```

`make test` downloads the control-plane binaries for `ENVTEST_K8S_VERSION` into
`./bin` and exports `KUBEBUILDER_ASSETS` automatically - no hard-coded,
arch-specific asset path, so it works on arm64/macOS and amd64/Linux alike.

Because envtest has no kubelet, any test resource you create must still be valid
to the apiserver. For example, a `StatefulSet`/`Deployment` pod template must
declare at least one container, or the apiserver rejects it with a `422`.

## 3. End-to-end (kind)

`integration-tests/main.sh` builds the manager image, loads it into a
[kind](https://kind.sigs.k8s.io) cluster, deploys the operator via
`config/default`, applies the `Broker`/`MeshSync` samples, and asserts that the
broker StatefulSet and meshsync Deployment become ready and that the Broker CR
`status.endpoint` is populated.

```bash
make integration-tests          # full cycle: setup, assert, cleanup
# or step by step:
make integration-tests-setup
make integration-tests-run
make integration-tests-cleanup
```

### Fast local iteration (build → deploy → test → repeat)

```bash
make e2e-dev                    # reuse the kind cluster, rebuild + reload +
                                # restart the operator, re-run all assertions
```

Environment knobs (all optional):

| Variable | Effect |
|----------|--------|
| `REUSE_CLUSTER=1` | Keep the existing kind cluster; reload images and restart the workloads instead of recreating everything. |
| `MESHSYNC_VERSION=<tag>` | Pin the MeshSync CR to `meshery/meshsync:<tag>`. Point it at a locally built tag to test cross-repo meshsync changes end-to-end before they are published (locally present images are side-loaded, not pulled). |
| `NO_CACHE=1` | Force a cold operator image build (layer cache is used by default). |
| `KIND_NODE_IMAGE=<image>` | Pin the kind node (Kubernetes) version; CI pins this explicitly. |
| `CERT_MANAGER_VERSION=<ver>` | Override the pinned cert-manager release used for the conversion webhook. |

The harness pre-loads the workload images into the kind node so pod startup is
not gated on first-time image pulls, and it uses portable shell (works on GNU
and BSD/macOS). Beyond workload readiness, the assertions verify the contract
that matters: `BROKER_URL` carries no literal credential (token flows through a
`secretKeyRef`), and MeshSync holds a **live client connection on the NATS
server** (checked via the broker's `connz` monitoring endpoint) — deployments
being Available proves nothing about broker connectivity.

> The current e2e harness is bash-on-kind with a single Service-type path. A
> richer matrix (ClusterIP/NodePort/LoadBalancer, networking reconfiguration,
> conversion/upgrade, finalizer cleanup, leader election) and CI promotion are
> delivered in WS-7 (#789).

## Conventions

- After changing API types or `+kubebuilder` markers, run `make manifests
  generate` and commit the result; CI fails if the generated output is stale.
- New behavior gets a test case in the existing suite for the affected package
  rather than a brand-new test file.
