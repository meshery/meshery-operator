# OLM bundle regeneration (WS-1)

The committed bundle under `bundle/0.0.1/` is **stale**: it was generated in
2022 against the old Kubebuilder `v2` layout and operator-sdk `v1.14.0`, and its
embedded install spec still reflects the pre-hardening manager (wildcard
ClusterRole, `kube-rbac-proxy:v0.5.0`, `hostPort`, `30Mi` memory limit). None of
that matches the current `config/` (least-privilege RBAC, hardened manager,
Kubebuilder `v4` layout).

It must be **regenerated**, not hand-edited, so that the bundle stays a faithful
projection of `config/`. Regeneration requires the `operator-sdk` CLI
(>= v1.42), which is not available in every contributor environment:

```bash
# Pick the next semver and regenerate from the v4 layout + current config/.
make bundle VERSION=0.1.0
operator-sdk bundle validate ./bundle
```

When regenerating, also apply the following metadata that the frozen bundle
lacks (these are the remaining WS-1 acceptance items):

- **Version**: bump from the frozen `0.0.1` (e.g. `0.1.0`) — set via
  `make bundle VERSION=…`.
- **Upgrade graph**: add `replaces: meshery-operator.v0.0.1` and
  `olm.skipRange: '>=0.0.1 <0.1.0'` to the ClusterServiceVersion so OLM can
  order upgrades.
- **Install modes**: broaden beyond `AllNamespaces` only if/when the manager
  honours a watched-namespace scope (`WATCH_NAMESPACE`); do not claim
  `SingleNamespace`/`MultiNamespace` support the controller does not implement.
- **Provenance stamps**: `operators.operatorframework.io/builder` and
  `operators.operatorframework.io/project_layout` (and the matching keys in
  `metadata/annotations.yaml`) are written by operator-sdk at regeneration time;
  they should not be edited by hand.

The maintainer-email typo (`urakiny@gmai.com` → `urakiny@gmail.com`) has already
been corrected in place because it is independent of regeneration.
