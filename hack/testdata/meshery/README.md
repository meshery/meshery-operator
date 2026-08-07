# Fixture: a meshery/meshery checkout

A verbatim copy of the chart files under `install/kubernetes/helm/` in
[`meshery/meshery`](https://github.com/meshery/meshery) master, trimmed to the
files `hack/sync-downstream.sh` reads or rewrites. `hack/sync_downstream_test.go`
copies this tree into a temporary directory and runs a full release stamp against
it.

It is a copy of the real thing on purpose. The bug this fixture guards against
(meshery/meshery-operator#878) was a stamp whose file set was incomplete, and a
hand-written sketch of the chart would have "passed" while the real chart drifted
- the fixture has to contain the files that actually exist, with the values they
actually carry, or it proves nothing.

## Refreshing it

Re-copy from `meshery/meshery` master whenever that chart's shape changes - a new
subchart, a renamed key, a reformatted README row:

```bash
B=https://raw.githubusercontent.com/meshery/meshery/master/install/kubernetes/helm
D=hack/testdata/meshery/install/kubernetes/helm
for f in meshery-operator/Chart.yaml \
         meshery-operator/values.yaml \
         meshery-operator/README.md \
         meshery-operator/charts/meshery-broker/Chart.yaml \
         meshery-operator/charts/meshery-broker/README.md \
         meshery-operator/charts/meshery-meshsync/Chart.yaml \
         meshery-operator/charts/meshery-meshsync/README.md \
         meshery/Chart.yaml; do
  curl -sSfL "$B/$f" -o "$D/$f"
done
go test ./hack/...
```

Then update `fixtureVersion` in `hack/sync_downstream_test.go` to whatever
release the refreshed tree advertises.

## What is deliberately absent

- `meshery/Chart.lock` and `meshery/charts/*.tgz` - the test writes its own, at
  the version it stamps, so the script's `helm dependency update` branch is
  skipped. That keeps the tests hermetic: no `helm` on `PATH`, no network. The
  skip is the script's own documented behaviour for an already-vendored version,
  not a test-only bypass.
- Chart `templates/`, `crds/` and `files/` - the stamp does not read them, and
  the CRD bundle it copies in is supplied by the test.
