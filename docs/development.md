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

This section serves two different populations - a reader whose file is already gone, and a
reader who has not pulled yet and whose copy has drifted - and advice that is safe for one
destroys the other's data. Earlier drafts tried to route the two apart with a file-based
test, and no such test exists: untracking is precisely what removes the file from git's
view, so after the pull a copy a Claude Code session recreated is indistinguishable from
one you never pulled over. The section is therefore built so that guessing wrong is
harmless, rather than so that you have to guess right.

**Which side of the pull are you on?** History answers it, and a recreated file cannot
fool history:

```bash
git log -1 --oneline -- .claude/settings.local.json
```

That prints the subject of the last commit to touch the file. `chore: untrack
settings.local.json, promote shared hooks to settings.json` means you have already pulled
the change. The subject that *added* the file means you have not: pull first - the pull is
what deletes the file - then recover below, or follow the note at the end of this section
if the pull aborts. Treat this as a hint about what to expect, not a gate; no command below
is conditional on it, and the guarded move is what protects you if it is wrong, so a wrong
answer costs you a confusing paragraph and nothing else.

**Recover the pre-split copy.** This reads it from the parent of the commit that deleted
the file, so it does not depend on reflog position, and it writes a sidecar rather than the
live path, so it is safe to run in any state:

```bash
git show "$(git rev-list -1 HEAD -- .claude/settings.local.json)^:.claude/settings.local.json" > ~/meshery-operator-settings.local.json.recovered
```

The sidecar lands in your home directory for two reasons: outside the repo, because
`.gitignore` covers the exact path and not a `.recovered` suffix, so a routine `git add -A`
would otherwise stage your `permissions` and `enabledMcpjsonServers` into someone else's PR
- the exact leak the split exists to close; and outside `/tmp`, because a predictable name
in a world-writable directory is a symlink target, and a planted symlink turns `>` into a
write primitive aimed at whatever it points to. The name is repo-scoped for the same reason
the backup below is.

Read it before you do anything with it:

```bash
cat ~/meshery-operator-settings.local.json.recovered
```

**Put it in place.** Both guards are mechanical protection rather than defensive habit, and
they cover different failures: `[ ! -e ]` stops the move overwriting a live file, which is
what makes guessing wrong above harmless, and `[ -s ]` stops a failed `git show` installing
an empty sidecar as a 0-byte `.claude/settings.local.json` that Claude Code cannot parse.
Keep both.

```bash
[ ! -e .claude/settings.local.json ] && [ -s ~/meshery-operator-settings.local.json.recovered ] && mv ~/meshery-operator-settings.local.json.recovered .claude/settings.local.json
```

If a live `.claude/settings.local.json` exists - a session recreated it after your pull, or
you have not pulled yet - the move is a no-op by design. Your current file is untouched and
the sidecar holds the last tracked copy: merge across whatever per-machine keys you want by
hand, then `rm ~/meshery-operator-settings.local.json.recovered`. There is deliberately no
command here that copies the sidecar over an existing file.

If your copy was untouched since you cloned, what you recovered is byte-identical to what
you had. If a Claude Code session had rewritten it - the abort case below, which you may
have already resolved yourself with `git checkout --`, `git stash`, or `git reset` - you
get the last tracked version instead, so re-add any per-machine keys that changed after
that point. If you resolved it with `git stash`, do not retype them: the stash still holds
your drifted copy verbatim, so read them back off `git stash show -p`.

**Then prune the `hooks` block** from the local file: the entire `"hooks": { ... }` object,
including any dead `tools/hooks/helm-chart-audit.py` `PostToolUse` entry your copy carries.
Order matters - recovering a whole pre-split copy reinstates the stale block, so pruning
before the file is back in place is undone. Those registrations now come from the tracked
`.claude/settings.json`, which a local `hooks` block does not override but merges into
additively: every promoted hook fires twice, giving doubled SessionStart output and
duplicate deny reasons with no obvious cause.

Keep everything else - `enabledMcpjsonServers`, `disabledMcpjsonServers`,
`additionalDirectories`, and `permissions` are per-machine and belong in the local file.

> **If the pull aborts** with "Your local changes to the following files would be
> overwritten by merge", you are the drifted reader by definition - that abort only happens
> when your copy differs from the tracked one - and the refused merge leaves your file
> untouched on disk, so the drift is still there to save. Back it up first. That backup is
> not optional and the recovery above is not a substitute for it: recovery yields the last
> *tracked* version, and the backup is the only thing preserving yours. Keep the name
> repo-scoped - sibling repos ship this same note, and a shared filename would restore one
> clone's settings into another:
>
> ```bash
> cp .claude/settings.local.json ~/meshery-operator-settings.local.json.bak
> ```
>
> Then discard the working copy, pull, and restore - all three steps, not the first two:
>
> ```bash
> [ -s ~/meshery-operator-settings.local.json.bak ] && git checkout -- .claude/settings.local.json && git pull
> cp ~/meshery-operator-settings.local.json.bak .claude/settings.local.json
> ```
>
> That guard is mechanical too, not defensive habit: it refuses to discard your drift until
> the backup is really on disk and non-empty, so arriving here without having run the `cp`
> above costs you a command that does nothing rather than your settings. Then prune the
> `hooks` block from the restored file exactly as described above.

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
