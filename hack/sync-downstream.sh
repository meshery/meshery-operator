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
#   2. Stamps EVERY file in the operator chart that names the operator release:
#      the parent Chart.yaml version/appVersion, values.yaml image.tag, the
#      parent README's Version/AppVersion badges and image.tag row, and each
#      subchart's Chart.yaml appVersion + README AppVersion badge. See "The
#      stamped file set" below.
#   3. Bumps the parent meshery chart's meshery-operator dependency to <version>
#      and re-vendors it (helm dependency update: Chart.lock + charts/*.tgz).
#   4. Removes the legacy duplicate CRD copy at meshery/crds/crds.yaml (the
#      operator chart is the single source for operator CRDs).
#
# The stamped file set
#
# Every file that advertises the operator release must move in the SAME pass.
# It used to be only the parent Chart.yaml and values.yaml, and everything left
# out drifted exactly as you would predict: both subcharts sat on the moving tag
# `stable-latest` for an unknown number of releases, and the chart README still
# advertised 1.0.0 while the Chart.yaml beside it read 1.0.5. A published Helm
# archive is immutable, so an appVersion it advertises that is a moving tag names
# a different application from one publish to the next - and a README that
# disagrees with values.yaml is simply wrong about what the chart installs
# (meshery/meshery-operator#878).
#
# The subcharts deploy no workload of their own; each ships one custom resource
# (Broker, MeshSync) whose images the operator's controllers choose. So the only
# honest reading of a subchart appVersion is "the operator release whose
# controllers reconcile this CR" - the same value the parent gets. meshery/meshery
# asserts that agreement in CI (install/scripts/check-operator-chart-appversions.sh);
# this stamp is what keeps it true across releases.
#
# The parent README is rewritten value-by-value rather than regenerated: it
# carries hand-written prose that `helm-docs` would delete, because values.yaml
# has no `# --` description comments and the chart has no README.md.gotmpl. When
# that migration lands downstream, the README block here collapses to a
# regenerate step. Its header comment records the same constraint. The subchart
# READMEs are pure helm-docs output and could be regenerated, but nothing runs
# the generator on a release, so their AppVersion badge is stamped too rather
# than left to rot behind the Chart.yaml beside it.
#
# Every substitution below is asserted afterwards (see `assert_stamped`). A stamp
# that silently matches nothing is precisely the failure this script exists to
# prevent, so a pattern that stops matching fails the release sync loudly.
#
# Requirements: helm on PATH; run from the meshery-operator repo root after
# `make crds`.
set -euo pipefail

MESHERY_DIR="${1:?usage: hack/sync-downstream.sh <meshery-checkout-path> <version>}"
VERSION="${2:?usage: hack/sync-downstream.sh <meshery-checkout-path> <version>}"
case "$VERSION" in
  v*) echo "error: version must be bare (1.2.3), got '$VERSION'" >&2; exit 1 ;;
esac
# A moving channel tag must never reach a published chart: the archive is
# immutable but the tag is not. Same guard, same reason, as the sibling
# hack/stamp-operator-version.sh applies to the in-repo artifacts.
if ! printf '%s' "$VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
  echo "error: '$VERSION' is not a semantic version; a chart must never advertise a moving tag" >&2
  exit 1
fi

# CRDS_SRC is overridable so the test suite can drive a full stamp against a
# fixture checkout without a `make crds` render in the working tree.
CRDS_SRC="${CRDS_SRC:-dist/crds.yaml}"
[ -f "$CRDS_SRC" ] || { echo "error: $CRDS_SRC not found — run 'make crds' first" >&2; exit 1; }

OPERATOR_CHART="$MESHERY_DIR/install/kubernetes/helm/meshery-operator"
PARENT_CHART="$MESHERY_DIR/install/kubernetes/helm/meshery"
[ -d "$OPERATOR_CHART" ] || { echo "error: $OPERATOR_CHART missing — is $MESHERY_DIR a meshery/meshery checkout?" >&2; exit 1; }

# 1. CRDs: install-time copy + the Job-consumed copy. Byte-identical by construction.
mkdir -p "$OPERATOR_CHART/crds" "$OPERATOR_CHART/files"
cp "$CRDS_SRC" "$OPERATOR_CHART/crds/crds.yaml"
cp "$CRDS_SRC" "$OPERATOR_CHART/files/crds.yaml"

# VERSION_RE is $VERSION with its dots escaped, for the assertions below: an
# unescaped 1.0.5 is an ERE that also matches 1x0y5.
VERSION_RE="$(printf '%s' "$VERSION" | sed 's/\./\\./g')"

# assert_stamped fails the sync when a file the stamp claims to have rewritten
# does not actually carry $VERSION afterwards. Without it a pattern that stops
# matching - a reformatted README row, a renamed key - is a silent no-op, which
# is how the subcharts and the READMEs drifted in the first place.
assert_stamped() {
  local file="$1" description="$2" pattern="$3"
  if ! grep -Eq -- "$pattern" "$file"; then
    echo "error: $file: $description was not stamped to $VERSION" >&2
    echo "       the stamp pattern no longer matches this file; fix hack/sync-downstream.sh" >&2
    exit 1
  fi
}

# stamp_app_version rewrites a Chart.yaml's top-level appVersion, appending the
# key when the chart has none. `version:` is never touched here: the parent's is
# stamped separately below, and a subchart's tracks its own lifecycle (0.5.0).
stamp_app_version() {
  local chart_yaml="$1" description="$2"
  if grep -q '^appVersion:' "$chart_yaml"; then
    perl -pi -e "s/^appVersion: .*/appVersion: \"$VERSION\"/" "$chart_yaml"
  else
    printf 'appVersion: "%s"\n' "$VERSION" >> "$chart_yaml"
  fi
  assert_stamped "$chart_yaml" "$description" "^appVersion: \"${VERSION_RE}\"\$"
}

# stamp_app_version_badge rewrites the helm-docs `AppVersion` badge in a chart
# README. The `Version` badge beside it is deliberately left alone - on a
# subchart it advertises the subchart's own version, and on the parent it is
# stamped explicitly below. The substitution anchors on the leading `![` so the
# AppVersion pattern cannot also match the Version badge, whose label is a
# suffix of it.
#
# A chart with no README is not an error; a README whose badge stopped matching
# is, which is what the assertion covers.
stamp_app_version_badge() {
  local readme="$1"
  [ -f "$readme" ] || return 0
  VERSION="$VERSION" perl -pi -e '
    my $v = $ENV{VERSION};
    s{!\[AppVersion: [^\]]*\]\(https://img\.shields\.io/badge/AppVersion-[^-)]*-}{![AppVersion: $v](https://img.shields.io/badge/AppVersion-$v-}g;
  ' "$readme"
  assert_stamped "$readme" "AppVersion badge" \
    "!\[AppVersion: ${VERSION_RE}\]\(https://img\.shields\.io/badge/AppVersion-${VERSION_RE}-"
}

# 2. Parent chart version + appVersion. Only rewrite the top-level keys.
perl -pi -e "s/^version: .*/version: $VERSION/" "$OPERATOR_CHART/Chart.yaml"
assert_stamped "$OPERATOR_CHART/Chart.yaml" "chart version" "^version: ${VERSION_RE}\$"
stamp_app_version "$OPERATOR_CHART/Chart.yaml" "chart appVersion"

# Pin the manager image tag explicitly (2-space-indented `tag:` under image:).
# helm-chart-releaser re-stamps appVersion with the *Meshery Server* tag when
# it republishes charts at server releases, so an appVersion-derived image tag
# would point at a nonexistent operator image under that publish path.
perl -pi -e "s/^  tag: \"[^\"]*\"/  tag: \"$VERSION\"/" "$OPERATOR_CHART/values.yaml"
assert_stamped "$OPERATOR_CHART/values.yaml" "image.tag" "^  tag: \"${VERSION_RE}\"\$"

# 2b. Subchart appVersion + the AppVersion badge in each subchart README (pure
#     helm-docs output, so the badge is regenerable in principle but drifts until
#     someone runs the generator - stamping it costs one line and closes that).
#     Subcharts are discovered from the directories rather than named, so a third
#     one added downstream is stamped the day it appears instead of passing
#     unexamined.
# A space-delimited string rather than an array: `${arr[*]}` on an EMPTY array
# is an unbound-variable error under `set -u` in bash 3.2, which ships as
# /bin/bash on macOS - so the zero-subchart case would abort with a confusing
# error instead of the dependency message below, which is the one that explains
# what actually went wrong.
STAMPED_SUBCHARTS=" "
for subchart_yaml in "$OPERATOR_CHART"/charts/*/Chart.yaml; do
  [ -f "$subchart_yaml" ] || continue
  stamp_app_version "$subchart_yaml" "subchart appVersion"
  stamp_app_version_badge "$(dirname "$subchart_yaml")/README.md"
  STAMPED_SUBCHARTS+="$(basename "$(dirname "$subchart_yaml")") "
done

# Cross-check the discovered set against the parent's declared dependencies: a
# dependency that is not unpacked as a directory is never reached by the loop
# above, and would drift silently. `- name:` entries also appear under
# `maintainers:` at the same indentation, so the scan is scoped to the
# dependencies block and ends at the next top-level key.
while read -r dep; do
  [ -n "$dep" ] || continue
  case "$STAMPED_SUBCHARTS" in
    *" $dep "*) ;;
    *) echo "error: $OPERATOR_CHART declares dependency '$dep' but charts/$dep/Chart.yaml was not stamped" >&2
       echo "       only an unpacked subchart directory can be stamped; a vendored .tgz cannot be" >&2
       exit 1 ;;
  esac
done < <(awk '
  /^dependencies:/ { in_deps = 1; next }
  in_deps && /^[^[:space:]#]/ { in_deps = 0 }
  in_deps && $1 == "-" && $2 == "name:" { print $3 }
' "$OPERATOR_CHART/Chart.yaml")

# 2c. The parent chart README: both badges and the image.tag row. It is
#     helm-docs output too, but NOT regenerable today (see "The stamped file set"
#     above), so its values are rewritten in place - the same value-level rewrite
#     values.yaml gets. Missing is a hard error here, unlike a subchart README:
#     this file is what the chart's readers see, and it advertises the release.
#     VERSION goes through the environment because the values row contains
#     backticks, which a double-quoted shell string would try to run.
README="$OPERATOR_CHART/README.md"
[ -f "$README" ] || {
  echo "error: $README missing - the chart README advertises the release and must be stamped with it" >&2
  exit 1
}
VERSION="$VERSION" perl -pi -e '
  my $v = $ENV{VERSION};
  s{!\[Version: [^\]]*\]\(https://img\.shields\.io/badge/Version-[^-)]*-}{![Version: $v](https://img.shields.io/badge/Version-$v-}g;
  s{^\| image\.tag \| string \| `"[^"]*"`}{| image.tag | string | `"$v"`};
' "$README"
stamp_app_version_badge "$README"
assert_stamped "$README" "Version badge" \
  "!\[Version: ${VERSION_RE}\]\(https://img\.shields\.io/badge/Version-${VERSION_RE}-"
assert_stamped "$README" "image.tag row" \
  "^\| image\.tag \| string \| \`\"${VERSION_RE}\"\`"

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
