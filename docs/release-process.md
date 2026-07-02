# Release process

Internal notes for maintainers: how a meshery-operator release is cut, what
automation fires, how the release propagates into `meshery/meshery` and to
running deployments, and the invariants that keep the pipeline honest.

## Cutting a release

1. `release-drafter.yml` maintains a draft release on every master push
   (`.github/release-drafter.yml` computes the next patch version; token
   `RELEASE_NOTES_PAT`).
2. A maintainer edits the draft as needed and **publishes** it. Publishing
   creates the `v*` tag — that is the only manual step.

## What fires on publish

| Workflow | What it does |
|---|---|
| `multi-platform.yml` | Builds and pushes the multi-arch manager image (`linux/amd64,linux/arm64`) with tags `<version>` (semver), `stable-<tag>`, `stable-<sha>`, `stable-latest`; signs it with cosign (keyless). |
| `sbom.yml` | Attaches an SPDX SBOM to the release. |
| `sync-downstream.yml` | Steps 1–3 below. |

The CRD bundle assets (`crds.yaml`, `crds-webhook-conversion.yaml`) are
attached to the **draft** release by `release-drafter.yml` on every master
push — releases here can publish **immutable** (v1.0.0 and v1.0.1 both did),
and immutable releases reject asset uploads after publish, so the draft is
the only reliable attach point.

`sync-downstream.yml` (also runnable via `workflow_dispatch` with
`release-ver` to re-sync an existing tag):

1. **Release assets (best-effort backfill)** — renders `make crds` and
   attempts to upload both bundles to the release; on an immutable release
   this warns and continues (the canonical asset path is the draft attach
   above — this step only backfills still-mutable releases and must never
   block the sync below).
2. **Downstream sync** — checks out `meshery/meshery` and runs
   `hack/sync-downstream.sh`, which updates the `meshery-operator` chart's
   `crds/crds.yaml` + `files/crds.yaml`, stamps the chart's
   `version`/`appVersion`/`values.yaml image.tag` to the released version,
   bumps the parent `meshery` chart's dependency, and re-vendors it
   (`Chart.lock` + `charts/meshery-operator-<version>.tgz`). The result is
   committed as `l5io <ci@meshery.io>` with `--signoff` and pushed to master
   (same convention as `error-ref-publisher.yaml`). If the push is rejected
   (e.g. branch protection), it opens an automated PR instead.
3. **Operator-versioned chart publish (OCI)** — packages the just-synced
   chart at the operator's version and pushes it to
   `oci://ghcr.io/meshery/charts/meshery-operator`. Consumers:
   `helm install meshery-operator oci://ghcr.io/meshery/charts/meshery-operator --version <version>`.
   The push is best-effort (a registry-permission failure warns without
   failing the sync).

## Two chart version streams — two channels

- **Server-stamped, on meshery.io/charts** (pre-existing): `meshery/meshery`'s
  `helm-chart-releaser.yml` republishes every chart under
  `install/kubernetes/helm/` at each **Meshery Server** release, stamping
  `chart_version`/`app_version` with the *server* tag (v-prefixed). Meshery
  Server's meshkit deployment path (`ApplyHelmChart{Chart: "meshery-operator",
  Version: <server release>}`) looks up exactly these. **Never remove this
  path** — meshkit's lookup depends on it.
- **Operator-versioned, on ghcr.io** (this pipeline): pushed as OCI artifacts
  for standalone `helm install` consumers and version-pinned deployments.
  These deliberately do NOT go into the shared meshery.io/charts index:
  helm's semver treats the index's historical server-stamped versions
  (`v1.0.1`, `v1.0.50`, …) and operator versions (`1.0.1`, …) as the same
  version space, and `helm repo index --merge` silently drops colliding
  entries — verified empirically when operator `1.0.1` collided with the
  historical server-stamped `v1.0.1`.

Because the server-stamped publish rewrites `appVersion` with the server tag,
the chart's manager image tag is pinned **explicitly** in `values.yaml`
(`image.tag`, stamped by the sync script) rather than derived from
`appVersion` — an appVersion-derived tag would point at a nonexistent
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
  reconciliation (`UpgradeIfInstalled: true`) — they are not steady state.
- **The next Meshery release**: nothing to do beyond the automated sync
  commit already being on `meshery/meshery` master. The server release
  pipeline snapshots the chart as-is. Hygiene: bump the
  `github.com/meshery/meshery-operator` pin in `meshery/meshery`'s `go.mod`
  when the typed client surface changed, and note the bundled operator
  version in the server release notes (the chart's `appVersion` on master at
  cut time is the source of truth).
- Note the deliberate behavior change from the pre-1.0 chart: the image is a
  pinned version, not `stable-latest` + `pullPolicy: Always`, so operator
  updates are explicit and versioned — never a silent drift on pod restart.

## Invariants

- **Conversion strategy None ⇔ field-identical schemas.** The chart ships the
  plain CRD bundle (strategy `None`), which is exact only while `v1alpha1`
  and `v1alpha2` are field-identical. When the schemas diverge, flip the
  chart to webhook conversion (`webhook.enabled=true` default) — the
  instruction lives in `api/v1alpha1/conversion.go`.
- **CRD updates flow through the chart's update Job**, not Helm's `crds/`
  directory (which Helm applies only on first install). Disabling the
  webhook must reset conversion to `None` explicitly (the Job does this).
- **Immutable releases**: releases in this repo publish immutable (verified
  empirically on both `v1.0.0` and `v1.0.1`, the latter after the setting was
  believed disabled — treat immutability as the operative assumption).
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
3. Watch the three release workflows; confirm the l5io sync commit landed on
   `meshery/meshery` master (or merge the fallback PR).
4. Spot-check the OCI chart:
   `helm pull oci://ghcr.io/meshery/charts/meshery-operator --version <version>`.
5. If the typed client changed, follow up with the `go.mod` bump in
   `meshery/meshery`.
