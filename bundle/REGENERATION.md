# OLM bundle regeneration

The committed bundle under `bundle/` is a **generated projection of `config/`**,
produced by `operator-sdk` from the Kubebuilder `v4` layout (`bundle/manifests`,
`bundle/metadata`, `bundle/tests`). It is regenerated, not hand-edited, so it
stays faithful to `config/`.

The WS-1 regeneration is done: the frozen 2022 bundle (`bundle/0.0.1/`, Kubebuilder
`v2`, operator-sdk `v1.14.0`, wildcard ClusterRole, `kube-rbac-proxy:v0.5.0`,
`hostPort`, `30Mi` memory limit) no longer exists. The current CSV is
`meshery-operator.v0.1.0`, built by operator-sdk `v1.42.3` against
`go.kubebuilder.io/v4`, and it carries the upgrade graph
(`replaces: meshery-operator.v0.0.1`, `olm.skipRange: '>=0.0.1 <0.1.0'`) and the
hardened manager install spec.

## Regenerating

Requires the `operator-sdk` CLI (>= v1.42), which the Makefile does **not**
install and CI does not have - which is why bundle regeneration is not part of
the `manifests-drift` gate.

```bash
make bundle                    # regenerates from config/ at the committed CSV version
operator-sdk bundle validate ./bundle
```

The Makefile's `VERSION` default *is* the committed CSV version, so a bare
`make bundle` reproduces the bundle in place. `VERSION=<x.y.z>` renders a one-off
at some other version; a permanent bump belongs in that default, not on the
command line, or the next bare regeneration rewrites the CSV backwards.

`make bundle` stamps `IMG` (that is, `OPERATOR_RELEASE_VERSION`) into the CSV's
manager image via `kustomize edit set image`, so regenerate **after** any
release stamp rather than before, or the CSV will disagree with `config/`.

When bumping the CSV version, keep the upgrade graph honest: set `replaces` to
the previous CSV and widen `olm.skipRange` to cover it.

## What may change without operator-sdk

Only verbatim mirrors of what `config/` already says, never an independent edit:

- **The manager image**, which `hack/stamp-operator-version.sh` rewrites in
  place on every release (see [docs/release-process.md § Pinned
  images](../docs/release-process.md#pinned-images)). That textual stamp is
  byte-identical to what `make bundle` would write for that field, and it exists
  because the bundle must never be left naming a moving tag just because CI
  cannot run operator-sdk.
- **A field the CSV mirrors from `config/manager`** (the manager's
  `imagePullPolicy` moved with the pin this way) and **generated CRD text in
  `bundle/manifests/meshery.io_*.yaml` mirrored from `config/crd/bases`** (an
  API field's `description` after a doc-comment change). Copy the generated
  output exactly, then confirm the remaining bundle-vs-`config/` delta is
  unchanged - a hand-sync that is not a verbatim copy needs a real
  regeneration instead.

Everything else in the bundle - install spec structure, RBAC, the upgrade graph,
provenance stamps - comes from regeneration.

## Remaining metadata notes

- **Install modes**: broaden beyond `AllNamespaces` only if/when the manager
  honours a watched-namespace scope (`WATCH_NAMESPACE`); do not claim
  `SingleNamespace`/`MultiNamespace` support the controller does not implement.
- **Provenance stamps**: `operators.operatorframework.io/builder` and
  `operators.operatorframework.io/project_layout` (and the matching keys in
  `metadata/annotations.yaml`) are written by operator-sdk at regeneration time;
  they should not be edited by hand.
