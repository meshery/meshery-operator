# Fixture: a meshery/meshery checkout

A verbatim copy of chart files from
[`meshery/meshery`](https://github.com/meshery/meshery) master's
`install/kubernetes/helm/`. `hack/sync_downstream_test.go` copies this tree into
a temporary directory and runs a full release stamp against it.

## What is here, and by what rule

**Every non-template file under `install/kubernetes/helm/meshery-operator/`** -
the parent chart's `Chart.yaml`, `Chart.lock`, `values.yaml`, `README.md` and
`.helmignore`, and the same for each subchart - plus `meshery/Chart.yaml`, which
carries the operator dependency the stamp bumps.

The rule is deliberately about *where a file lives*, not about *what the stamp
touches*. See "What is deliberately absent" for why that distinction is the
whole point of the fixture.

It is a copy of the real thing on purpose. The bug this fixture guards against
(meshery/meshery-operator#878) was a stamp whose file set was incomplete, and a
hand-written sketch of the chart would have "passed" while the real chart drifted
- the fixture has to contain the files that actually exist, with the values they
actually carry, or it proves nothing.

## Refreshing it

Re-copy from `meshery/meshery` master whenever that chart's shape changes - a new
subchart, a new top-level file, a renamed key, a reformatted README row:

```bash
B=https://raw.githubusercontent.com/meshery/meshery/master/install/kubernetes/helm
D=hack/testdata/meshery/install/kubernetes/helm
for f in meshery-operator/.helmignore \
         meshery-operator/Chart.lock \
         meshery-operator/Chart.yaml \
         meshery-operator/values.yaml \
         meshery-operator/README.md \
         meshery-operator/charts/meshery-broker/.helmignore \
         meshery-operator/charts/meshery-broker/Chart.yaml \
         meshery-operator/charts/meshery-broker/values.yaml \
         meshery-operator/charts/meshery-broker/README.md \
         meshery-operator/charts/meshery-meshsync/.helmignore \
         meshery-operator/charts/meshery-meshsync/Chart.yaml \
         meshery-operator/charts/meshery-meshsync/values.yaml \
         meshery-operator/charts/meshery-meshsync/README.md \
         meshery/Chart.yaml; do
  mkdir -p "$D/$(dirname "$f")"
  curl -sSfL "$B/$f" -o "$D/$f"
done
go test ./hack/...
```

A file that appears under the operator chart upstream and is not in that list
belongs in it - add it rather than deciding the stamp does not need it.

Then update `fixtureVersion` in `hack/sync_downstream_test.go` to whatever
release the refreshed tree advertises.

## What is deliberately absent

- Chart `templates/`, `crds/` and `files/` - large generated payloads. The stamp
  does not read `templates/`, and the CRD bundles it copies into `crds/` and
  `files/` are supplied by the test.
- `meshery/`'s files other than `Chart.yaml`, including `Chart.lock` and
  `charts/*.tgz` - the test writes its own, at the version it stamps, so the
  checkout starts out already vendored. `helm dependency update` is driven
  through the script's `HELM_BIN` override and answered by a stub that records
  the call, which keeps the tests hermetic (no `helm` on `PATH`, no network)
  while still letting them assert *whether* the script re-vendored.

The trim criterion is deliberately **not** "the files the stamp touches". That
criterion is the original gap restated: applied to the pre-fix script it would
have excluded both subchart `Chart.yaml`s and every README, and the whole-tree
sweep in `sync_downstream_test.go` would have reported the fixture clean while
the real chart drifted. Trimming by location instead leaves files in the fixture
that the stamp has no opinion about, which is exactly what lets the sweep
discover a drift site nobody has thought of yet.
