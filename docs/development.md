# Development

## Prerequisites

- **Go** - the version in [`go.mod`](../go.mod) (`go-version-file` is used in CI).
- **Docker** - for building the manager image and running the kind e2e suite.
- **kubectl**, and a cluster for manual testing (Docker Desktop, kind, or minikube).

All other tooling (`controller-gen`, `kustomize`, `setup-envtest`, `kind`,
`golangci-lint`, `opm`) is installed on demand into `./bin` by the Makefile at
pinned versions - you do not need them on your `PATH`.

## Claude Code agent configuration

`.claude/` is split by ownership:

- **`.claude/settings.json`** is tracked and shared. It registers the hook scripts in
  `.claude/hooks/` - including the no-AI-attribution guard that
  [AGENTS.md](../AGENTS.md) advertises - so a fresh clone gets every guard with no
  further setup. Shared, repo-level agent configuration belongs here.
- **`.claude/settings.local.json`** is git-ignored per-developer state
  (`enabledMcpjsonServers`, `disabledMcpjsonServers`, `additionalDirectories`,
  `permissions`). Claude Code rewrites it on every session, so tracking it makes an
  unrelated branch dirty on each run and risks a truncated copy landing in someone
  else's PR. Never re-add it to git.

### Migrating a clone made before the split

Untracking deletes the file: pulling that change removes `.claude/settings.local.json`
from your clone, along with the per-developer keys it holds.

**Only recover if `.claude/settings.local.json` is already gone.**

If it is still on disk you have not pulled yet - skip to the note at the end of this
section. The recovery command below deliberately lands in a private throwaway file,
because a shell applies `>` before git runs: aimed straight at the live path it would
truncate your settings to zero bytes and only then fail, since on a pre-split checkout
the lookup resolves to the commit that *added* the file and its parent holds no copy to
print. Do not retarget the redirect.

**Recover the deleted file.** This reads the copy from the parent of the commit that
deleted it, so it does not depend on reflog position:

```bash
RECOVERED=$(mktemp)
git show "$(git rev-list -1 HEAD -- .claude/settings.local.json)^:.claude/settings.local.json" > "$RECOVERED"
```

`mktemp` is used rather than a fixed name in `/tmp`: it creates an unpredictable path
readable only by you, so nothing else on the machine can be truncated through it and your
per-machine config is not left sitting in a shared directory. That matters because it is
the same destroy-by-redirect hazard the gate above guards against, merely aimed elsewhere
- an earlier draft of this guide hardened the live path and then reintroduced the trap by
recovering into a predictable `/tmp` name, where a planted symlink turns `>` into a write
primitive. Check that the result looks like your settings, then put it in place and clean
up:

```bash
[ -s "$RECOVERED" ] && cp "$RECOVERED" .claude/settings.local.json && rm -f "$RECOVERED"
```

The `[ -s ... ]` guard is load-bearing, not defensive habit: if the `git show` above failed
for any reason it leaves `$RECOVERED` empty, and an unguarded `cp` would then overwrite
your live settings with nothing - destroying the file this section exists to save.

If your copy was untouched since you cloned, that is byte-identical to what you had. If a
Claude Code session had rewritten it - the abort case below, which you may have already
resolved yourself with `git checkout --`, `git stash`, or `git reset` - you get the last
tracked version instead, so re-add any per-machine keys that changed after that point.

**Then prune the `hooks` block** from the recovered file: the entire `"hooks": { ... }`
object, including any dead `tools/hooks/helm-chart-audit.py` `PostToolUse` entry your copy
carries. Order matters - recovering a whole pre-split copy reinstates the stale block, so
pruning before recovering is undone. Those registrations now come from the tracked
`.claude/settings.json`, which a local `hooks` block does not override but merges into
additively: every promoted hook fires twice, giving doubled SessionStart output and
duplicate deny reasons with no obvious cause.

Keep everything else - `enabledMcpjsonServers`, `disabledMcpjsonServers`,
`additionalDirectories`, and `permissions` are per-machine and belong in the local file.

> **Not pulled yet?** A backup is optional - once the file is gone the recovery above
> works regardless - and it only preserves session drift the recovered copy cannot carry.
> Pull first, then recover. If you do want a backup, keep the name repo-scoped, because
> sibling repos ship this same note and a shared filename would restore one clone's
> settings into another:
> `cp .claude/settings.local.json ~/meshery-operator-settings.local.json.bak`
>
> If the pull aborts with "Your local changes to the following files would be overwritten
> by merge", discard your copy and pull again:
> `git checkout -- .claude/settings.local.json && git pull`.
>
> Restoring from that backup is `cp ~/meshery-operator-settings.local.json.bak
> .claude/settings.local.json` - use it instead of the recovery command above, then prune
> the `hooks` block exactly as described.

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
