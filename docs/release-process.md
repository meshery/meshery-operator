# Release process

Internal notes for maintainers: how a meshery-operator release is cut, what
automation fires, how the release propagates into `meshery/meshery` and to
running deployments, and the invariants that keep the pipeline honest.

## Cutting a release

1. `release-drafter.yml` maintains a draft release on every master push
   (`.github/release-drafter.yml` computes the next patch version; token
   `RELEASE_NOTES_PAT`).
2. A maintainer edits the draft as needed and **publishes** it. Publishing
   creates the `v*` tag - that is the only manual step.

## What fires on publish

| Workflow | What it does |
|---|---|
| `multi-platform.yml` | Builds and pushes the multi-arch manager image (`linux/amd64,linux/arm64`) with tags `<version>` (semver), `stable-<tag>`, `stable-<sha>`, `stable-latest`; signs it with cosign (keyless). |
| `sbom.yml` | Attaches an SPDX SBOM to the release. |
| `sync-downstream.yml` | Steps 1–3 below. |
| `stamp-release.yml` | Waits for `meshery/meshery-operator:<version>` to appear on Docker Hub, then advances master's pinned manager image to it (see [Pinned images](#pinned-images)). |

The CRD bundle assets (`crds.yaml`, `crds-webhook-conversion.yaml`) are
attached to the **draft** release by `release-drafter.yml` on every master
push - releases here can publish **immutable** (v1.0.0 and v1.0.1 both did),
and immutable releases reject asset uploads after publish, so the draft is
the only reliable attach point.

`sync-downstream.yml` (also runnable via `workflow_dispatch` with
`release-ver` to re-sync an existing tag):

1. **Release assets (best-effort backfill)** - renders `make crds` and
   attempts to upload both bundles to the release; on an immutable release
   this warns and continues (the canonical asset path is the draft attach
   above - this step only backfills still-mutable releases and must never
   block the sync below).
2. **Downstream sync** - checks out `meshery/meshery` and runs
   `hack/sync-downstream.sh`, which updates the `meshery-operator` chart's
   `crds/crds.yaml` + `files/crds.yaml`, stamps [the whole versioned file
   set](#the-stamped-chart-file-set) to the released version, bumps the parent
   `meshery` chart's dependency, and re-vendors it (`Chart.lock` +
   `charts/meshery-operator-<version>.tgz`). The result is
   committed as `l5io <ci@meshery.io>` with `--signoff` and pushed to master
   (same convention as `error-ref-publisher.yaml`). If the push is rejected
   (e.g. branch protection), it opens an automated PR instead.
3. **Operator-versioned chart publish (OCI)** - packages the just-synced
   chart at the operator's version and pushes it to
   `oci://ghcr.io/meshery/charts/meshery-operator`. Consumers:
   `helm install meshery-operator oci://ghcr.io/meshery/charts/meshery-operator --version <version>`.
   The push is best-effort (a registry-permission failure warns without
   failing the sync).

## The stamped chart file set

`hack/sync-downstream.sh` rewrites **every** file in `meshery/meshery`'s
operator chart that advertises an operator release, in one pass:

| File | What is stamped |
|---|---|
| `meshery-operator/Chart.yaml` | `version:`, `appVersion:` |
| `meshery-operator/values.yaml` | `image.tag` |
| `meshery-operator/README.md` | `Version` + `AppVersion` badges, the `image.tag` values row |
| `meshery-operator/charts/*/Chart.yaml` | `appVersion:` (the subchart's own `version:` is left alone - it tracks the subchart's lifecycle) |
| `meshery-operator/charts/*/README.md` | `AppVersion` badge (the subchart's own `Version` badge is left alone) |
| `meshery/Chart.yaml` + `Chart.lock` + `charts/*.tgz` | the `meshery-operator` dependency version and the vendored archive |

The archive is repackaged whenever the operator chart's **content** changed
during the run, not merely when the version moved. `helm package` folds in every
file in the table above, so a re-sync of a tag that is already vendored (the
`sync-downstream` workflow's `workflow_dispatch` path) would otherwise commit
freshly stamped sources beside `charts/meshery-operator-<version>.tgz` still
carrying the old ones - the same drift, relocated into the artifact users
install from, and invisible to a check that reads the sources. A genuine no-op
re-run changes nothing and still skips, so the lock timestamp and the archive
bytes do not churn.

The set is the point. It used to be the first two rows only, and everything
omitted drifted: both subcharts sat on the moving tag `stable-latest` for an
unknown number of releases, and the chart README advertised `1.0.0` beside a
`Chart.yaml` that read `1.0.5` (meshery/meshery-operator#878). A published Helm
archive is immutable, so an `appVersion` naming a moving tag advertises a
different application at every publish; a README that disagrees with
`values.yaml` is simply wrong about what the chart installs.

Neither subchart deploys a workload - each ships one custom resource (`Broker`,
`MeshSync`) whose images the operator's controllers choose. So the only honest
reading of a subchart `appVersion` is "the operator release whose controllers
reconcile this CR", which is the value the parent already gets.
`meshery/meshery` asserts that agreement in CI
(`install/scripts/check-operator-chart-appversions.sh`); this stamp is what
keeps the assertion true across releases.

Two properties make the set maintainable rather than another list to rot:

- **Subcharts are discovered, not named.** The script walks `charts/*/`, then
  cross-checks the result against the parent `Chart.yaml`'s declared
  dependencies, so a third subchart is stamped the day it appears and a
  dependency the walk cannot reach fails the sync instead of drifting.
- **Every substitution is asserted.** A pattern that silently matches nothing is
  exactly how the README drifted, so the script verifies each rewritten value
  afterwards and fails the release sync when one no longer matches. Each
  assertion pins the whole value it claims to have written - a badge through its
  trailing `-informational`, not just the version prefix - because an assertion
  that checks a prefix accepts a mangled remainder.

The two shields.io badges are rewritten by one helper parameterised on the
label, so their encoding rules live in a single place. Those rules: the message
segment runs to the colour separator (a version is not `-`-free, and a rewrite
that stopped at the first `-` appended a prerelease suffix on every pass instead
of replacing it), and a `-` belonging to the version is doubled inside the badge
path, which is how shields.io and `helm-docs` both encode it.

`hack/sync_downstream_test.go` drives the script against a verbatim copy of
meshery master's chart tree (`hack/testdata/meshery/`) and asserts that no file
in it still advertises the previous release. The fixture holds every
non-template file under the operator chart - a rule deliberately independent of
what the stamp touches, so the sweep can surface a drift site nobody has listed
yet instead of only re-confirming the known ones. Refresh it from
`meshery/meshery` master when the chart's shape changes.

The parent README is rewritten value-by-value rather than regenerated: it
carries hand-written prose that `helm-docs` would delete, because `values.yaml`
has no `# --` description comments and the chart has no `README.md.gotmpl`. When
that migration lands downstream, this part of the stamp collapses to a
regenerate step.

## Two chart version streams - two channels

- **Server-stamped, on meshery.io/charts** (pre-existing): `meshery/meshery`'s
  `helm-chart-releaser.yml` republishes every chart under
  `install/kubernetes/helm/` at each **Meshery Server** release, stamping
  `chart_version`/`app_version` with the *server* tag (v-prefixed). Meshery
  Server's meshkit deployment path (`ApplyHelmChart{Chart: "meshery-operator",
  Version: <server release>}`) looks up exactly these. **Never remove this
  path** - meshkit's lookup depends on it.
- **Operator-versioned, on ghcr.io** (this pipeline): pushed as OCI artifacts
  for standalone `helm install` consumers and version-pinned deployments.
  These deliberately do NOT go into the shared meshery.io/charts index:
  helm's semver treats the index's historical server-stamped versions
  (`v1.0.1`, `v1.0.50`, …) and operator versions (`1.0.1`, …) as the same
  version space, and `helm repo index --merge` silently drops colliding
  entries - verified empirically when operator `1.0.1` collided with the
  historical server-stamped `v1.0.1`.

Because the server-stamped publish rewrites `appVersion` with the server tag,
the chart's manager image tag is pinned **explicitly** in `values.yaml`
(`image.tag`, stamped by the sync script) rather than derived from
`appVersion` - an appVersion-derived tag would point at a nonexistent
operator image under the server-stamped stream.

## How a release reaches deployments

- **Existing Meshery deployments**: a running Meshery Server vX.Y.Z deploys
  the operator chart pinned to *its own* version X.Y.Z, whose content froze
  when that server version was released. New operator versions reach those
  clusters when the deployment's **Meshery Server is upgraded**: meshkit
  re-applies the operator chart at the new server version → `helm upgrade` →
  the chart's CRD update Job refreshes the CRDs and the Deployment rolls to
  the pinned image. Manual stopgaps (direct
  `helm upgrade --version <operator-version>` or `kubectl apply` of the
  release's `crds.yaml`) work but can be reverted by the server's
  reconciliation (`UpgradeIfInstalled: true`) - they are not steady state.
- **The next Meshery release**: nothing to do beyond the automated sync
  commit already being on `meshery/meshery` master. The server release
  pipeline snapshots the chart as-is. Hygiene: bump the
  `github.com/meshery/meshery-operator` pin in `meshery/meshery`'s `go.mod`
  when the typed client surface changed, and note the bundled operator
  version in the server release notes (the chart's `appVersion` on master at
  cut time is the source of truth).
- Note the deliberate behavior change from the pre-1.0 chart: the image is a
  pinned version, not `stable-latest` + `pullPolicy: Always`, so operator
  updates are explicit and versioned - never a silent drift on pod restart.

## Pinned images

**No artifact this repo ships may name a moving tag.** A channel tag
(`stable-latest`, `edge-latest`, `latest`) is re-pointed in place by the
publisher, so an artifact a user applied months ago silently starts pulling a
build it was never rendered against. That is not hypothetical: the
`stable-latest` manager tag advanced to a master build requiring webhook
serving certs, and every older chart - which mounts no such certs - crashlooped
on `open /tmp/k8s-webhook-server/serving-certs/tls.crt: no such file or
directory`. Pinned tags are immutable by convention, which is also what makes
`imagePullPolicy: IfNotPresent` correct for them (and side-loaded kind images
and air-gapped clusters work).

Three independent pins, three mechanisms:

| Pin | Where | Advanced by |
|---|---|---|
| Manager image | `Makefile` `OPERATOR_RELEASE_VERSION` → `IMG`, `config/manager/{manager.yaml,kustomization.yaml}`, `config/manifests/default.yaml`, `bundle/manifests/*.clusterserviceversion.yaml` | `stamp-release.yml` on publish (`hack/stamp-operator-version.sh`, or `make stamp-release OPERATOR_RELEASE_VERSION=<version>` by hand) |
| MeshSync image | `pkg/meshsync/resources.go` `defaultMeshSyncVersion` | `meshsync-version-bump.yml` opens a weekly bump PR when meshery/meshsync publishes a semantically newer release whose image is on Docker Hub (GitHub resolves "latest" by tag creation time, so the comparison is by version, and a pre-release or older backport is skipped with a notice) |
| NATS image | the vendored chart, `pkg/broker/manifests/nats.gen.yaml` | bump `NATS_CHART_VERSION` and re-run `make nats-manifests` (drift-gated by `nats-chart-drift.yml`) |

The user-facing half of these pins is documented downstream, not here:
`meshery/meshery`'s
`docs/content/en/guides/infrastructure-management/configuring-operator-meshsync-broker.md`
(published to docs.meshery.io) is the authoritative page for `MeshSync` and
`Broker` `spec.version`, and it states the default image tag, the pull-policy
rule, and the tag forms accepted. Changing any of those here -
`defaultMeshSyncVersion`, the policy in `pkg/utils/image.go`, the normalisation
rules - goes stale there, so update that page in the same wave.

Guard rails, so a regression cannot land quietly:

- `make stamp-release` refuses anything that is not a bare semver, so the
  manager pin cannot be set to a channel tag even by hand.
- `make docker-push` and `make docker-buildx` refuse to run while `IMG` is
  still the `OPERATOR_RELEASE_VERSION`-derived default, so a bare push cannot
  overwrite a released image with a local build. Pass an explicit
  `IMG=<registry>/<repo>:<tag>`; the release image itself is published by
  `multi-platform.yml`, never by hand. `make docker-build` (local only) and
  `make bundle` (which must stamp the pinned release into the CSV) keep the
  default.
- `stamp-release.yml` polls Docker Hub before stamping: master may only ever
  name an image that exists.
- The `manifests-drift` CI job greps `config/`, `bundle/`, and the `Makefile`
  for `:*-latest` image references.
- `TestDefaultMeshSyncVersionIsPinned` fails if the MeshSync default becomes a
  moving tag, a v-prefixed tag (the registry publishes bare semver only), or
  a release predating the health endpoints.

Registry convention worth remembering: `meshery/meshsync`, `meshery/meshery-operator`,
and `nats` all publish **bare-semver** image tags (`1.0.3`), while their git
tags and GitHub releases are **v-prefixed** (`v1.0.3`). A `spec.version` copied
straight from a release name would 404, so the operator normalises a leading
`v` off an otherwise-semver `spec.version` before building the image reference
(`pkg/utils.NormalizeImageTag`).

Master's manager pin therefore names the **previous** release until
`stamp-release.yml` lands its commit. That is deliberate: the alternative is
master naming an image that does not exist yet.

## Invariants

- **Conversion strategy None ⇔ field-identical schemas.** The chart ships the
  plain CRD bundle (strategy `None`), which is exact only while `v1alpha1`
  and `v1alpha2` are field-identical. When the schemas diverge, flip the
  chart to webhook conversion (`webhook.enabled=true` default) - the
  instruction lives in `api/v1alpha1/conversion.go`.
- **CRD updates flow through the chart's update Job**, not Helm's `crds/`
  directory (which Helm applies only on first install). Disabling the
  webhook must reset conversion to `None` explicitly (the Job does this).
- **Immutable releases**: releases in this repo publish immutable (verified
  empirically on both `v1.0.0` and `v1.0.1`, the latter after the setting was
  believed disabled - treat immutability as the operative assumption).
  Sealed releases reject asset uploads forever, so assets must land on the
  draft (`release-drafter.yml`), and any post-publish upload must be
  best-effort.
- **Storage-version migration debt**: clusters upgraded from `v1alpha1`
  storage keep `status.storedVersions: [v1alpha1, v1alpha2]`. Until a
  migration (rewrite stored objects, prune `storedVersions`) runs, `v1alpha1`
  must remain `served: true`.

## Release checklist

1. Merge everything intended for the release; CI green on master.
2. Publish the release draft (creates the tag).
3. Watch the four release workflows; confirm the l5io sync commit landed on
   `meshery/meshery` master (or merge the fallback PR).
4. Confirm `stamp-release.yml` advanced master's `OPERATOR_RELEASE_VERSION` to
   the new version (or merge its `operator-pin/v<version>` fallback PR). It
   waits up to 30 minutes for the image, so it finishes after the others.
5. Spot-check the OCI chart:
   `helm pull oci://ghcr.io/meshery/charts/meshery-operator --version <version>`.
6. If the typed client changed, follow up with the `go.mod` bump in
   `meshery/meshery`.
