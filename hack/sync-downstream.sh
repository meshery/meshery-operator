#!/usr/bin/env bash
# Sync generated operator release artifacts into a meshery/meshery checkout.
#
# Usage: hack/sync-downstream.sh <meshery-checkout-path> <version>
#   <meshery-checkout-path>  Path to a meshery/meshery working tree.
#   <version>                Bare operator release version, e.g. 1.0.0 (no leading v).
#
# What it does (idempotent — a second run produces no further changes):
#   1. Copies dist/crds.yaml (rendered by `make crds`) into the meshery-operator
#      chart's crds/ (Helm install-time path) and files/ (consumed by the chart's
#      CRD update Job via .Files.Get, which is what refreshes CRDs on upgrade).
#   2. Stamps the chart's version/appVersion and the chart's default image tag
#      expectations to <version>.
#   3. Bumps the parent meshery chart's meshery-operator dependency to <version>
#      and re-vendors it (helm dependency update: Chart.lock + charts/*.tgz).
#   4. Removes the legacy duplicate CRD copy at meshery/crds/crds.yaml (the
#      operator chart is the single source for operator CRDs).
#
# Requirements: helm on PATH; run from the meshery-operator repo root after
# `make crds`.
set -euo pipefail

MESHERY_DIR="${1:?usage: hack/sync-downstream.sh <meshery-checkout-path> <version>}"
VERSION="${2:?usage: hack/sync-downstream.sh <meshery-checkout-path> <version>}"
case "$VERSION" in
  v*) echo "error: version must be bare (1.2.3), got '$VERSION'" >&2; exit 1 ;;
esac

CRDS_SRC="dist/crds.yaml"
[ -f "$CRDS_SRC" ] || { echo "error: $CRDS_SRC not found — run 'make crds' first" >&2; exit 1; }

OPERATOR_CHART="$MESHERY_DIR/install/kubernetes/helm/meshery-operator"
PARENT_CHART="$MESHERY_DIR/install/kubernetes/helm/meshery"
[ -d "$OPERATOR_CHART" ] || { echo "error: $OPERATOR_CHART missing — is $MESHERY_DIR a meshery/meshery checkout?" >&2; exit 1; }

# 1. CRDs: install-time copy + the Job-consumed copy. Byte-identical by construction.
mkdir -p "$OPERATOR_CHART/crds" "$OPERATOR_CHART/files"
cp "$CRDS_SRC" "$OPERATOR_CHART/crds/crds.yaml"
cp "$CRDS_SRC" "$OPERATOR_CHART/files/crds.yaml"

# 2. Chart version + appVersion. Only rewrite the top-level keys.
perl -pi -e "s/^version: .*/version: $VERSION/" "$OPERATOR_CHART/Chart.yaml"
if grep -q '^appVersion:' "$OPERATOR_CHART/Chart.yaml"; then
  perl -pi -e "s/^appVersion: .*/appVersion: \"$VERSION\"/" "$OPERATOR_CHART/Chart.yaml"
else
  printf 'appVersion: "%s"\n' "$VERSION" >> "$OPERATOR_CHART/Chart.yaml"
fi
# Pin the manager image tag explicitly (2-space-indented `tag:` under image:).
# helm-chart-releaser re-stamps appVersion with the *Meshery Server* tag when
# it republishes charts at server releases, so an appVersion-derived image tag
# would point at a nonexistent operator image under that publish path.
perl -pi -e "s/^  tag: \"[^\"]*\"/  tag: \"$VERSION\"/" "$OPERATOR_CHART/values.yaml"

# 3. Parent chart dependency bump + re-vendor. The dependency block is matched by
#    the adjacent name key so only the meshery-operator entry's version changes.
#    helm dependency update leaves the repository-less adapter deps alone
#    ("Assuming it exists in the charts directory") and repackages only the
#    file://-sourced operator chart + Chart.lock. Skipped when already vendored
#    at $VERSION so re-runs don't churn the lock timestamp / tgz bytes.
perl -0pi -e "s/(name: meshery-operator\n(?:[^\n]*\n)*?\s*version: )[^\n]*/\${1}$VERSION/" "$PARENT_CHART/Chart.yaml"
if [ ! -f "$PARENT_CHART/charts/meshery-operator-$VERSION.tgz" ] \
   || ! grep -A2 'name: meshery-operator' "$PARENT_CHART/Chart.lock" 2>/dev/null | grep -q "version: $VERSION"; then
  rm -f "$PARENT_CHART"/charts/meshery-operator-*.tgz
  helm dependency update "$PARENT_CHART" >/dev/null
fi

# 4. Retire the legacy duplicate CRD copy in the parent chart.
rm -f "$PARENT_CHART/crds/crds.yaml"
rmdir "$PARENT_CHART/crds" 2>/dev/null || true

if git -C "$MESHERY_DIR" status --porcelain -- install/kubernetes/helm | grep -q .; then
  echo "sync: updated meshery charts to operator v$VERSION"
else
  echo "sync: no changes (already at v$VERSION)"
fi
