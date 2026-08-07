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
#      and re-vendors it (helm dependency update: Chart.lock + charts/*.tgz)
#      whenever anything under the operator chart actually changed.
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
# hack/stamp-operator-version.sh applies to the in-repo artifacts - keep the two
# patterns identical.
#
# This is the SemVer grammar for a release version, not a loose "digits, dots and
# a suffix" approximation. The loose form accepted 01.2.3, 1.2.3-01 and 1.2.3-. ,
# none of which is a version: each would be stamped into the chart and only fail
# later, opaquely, when helm or an image pull could not resolve it. Numeric
# identifiers therefore reject leading zeros, and every prerelease identifier
# must be non-empty. Build metadata (+...) is deliberately NOT accepted: this
# value also becomes a container image tag, and '+' is not legal in one.
SEMVER_RE='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)'
SEMVER_RE="$SEMVER_RE"'(-(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)'
SEMVER_RE="$SEMVER_RE"'(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?$'
if ! printf '%s' "$VERSION" | grep -Eq "$SEMVER_RE"; then
  echo "error: '$VERSION' is not a semantic version; a chart must never advertise a moving tag" >&2
  exit 1
fi

# CRDS_SRC is overridable so the test suite can drive a full stamp against a
# fixture checkout without a `make crds` render in the working tree.
CRDS_SRC="${CRDS_SRC:-dist/crds.yaml}"
[ -f "$CRDS_SRC" ] || { echo "error: $CRDS_SRC not found — run 'make crds' first" >&2; exit 1; }

# HELM_BIN is overridable so the test suite can observe the step-3 re-vendor
# decision without helm on PATH. The release path always resolves plain `helm`.
HELM_BIN="${HELM_BIN:-helm}"

OPERATOR_CHART="$MESHERY_DIR/install/kubernetes/helm/meshery-operator"
PARENT_CHART="$MESHERY_DIR/install/kubernetes/helm/meshery"
[ -d "$OPERATOR_CHART" ] || { echo "error: $OPERATOR_CHART missing — is $MESHERY_DIR a meshery/meshery checkout?" >&2; exit 1; }

# The operator chart's content as this run found it, for the step-3 re-vendor
# decision. Snapshotted before the CRD copy rather than after, because the CRD
# bundles are packaged into the archive too.
#
# Compare the chart against its own prior content, not against git HEAD: the
# sync workflow runs against a checkout that may already carry unrelated staged
# or unstaged edits, and a release-critical decision must not depend on ambient
# worktree state. Same reasoning, same shape, as the BEFORE/checksum() pair in
# the sibling hack/stamp-operator-version.sh.
chart_checksum() { find "$OPERATOR_CHART" -type f -exec shasum {} + | sort | shasum; }
CHART_BEFORE="$(chart_checksum)"

# 1. CRDs: install-time copy + the Job-consumed copy. Byte-identical by construction.
mkdir -p "$OPERATOR_CHART/crds" "$OPERATOR_CHART/files"
cp "$CRDS_SRC" "$OPERATOR_CHART/crds/crds.yaml"
cp "$CRDS_SRC" "$OPERATOR_CHART/files/crds.yaml"

# VERSION_RE is $VERSION with its dots escaped, for the assertions below: an
# unescaped 1.0.5 is an ERE that also matches 1x0y5.
VERSION_RE="$(printf '%s' "$VERSION" | sed 's/\./\\./g')"

# BADGE_VERSION is $VERSION as shields.io encodes it inside a static badge path.
# That path is label-message-colour, so a '-' belonging to the value itself must
# be doubled or the badge renders as the label "Version-1.0.6" beside the message
# "rc.1". helm-docs escapes it the same way (`replace "-" "--"`), so a stamped
# badge stays byte-identical to what regenerating the README would produce.
BADGE_VERSION="$(printf '%s' "$VERSION" | sed 's/-/--/g')"
BADGE_VERSION_RE="$(printf '%s' "$BADGE_VERSION" | sed 's/\./\\./g')"

# assert_stamped fails the sync when a file the stamp claims to have rewritten
# does not actually carry $VERSION afterwards. Without it a pattern that stops
# matching - a reformatted README row, a renamed key - is a silent no-op, which
# is how the subcharts and the READMEs drifted in the first place.
#
# A fourth argument narrows the haystack to a value already extracted from the
# file. A substitution that targets one entry among near-identical neighbours
# cannot be checked by searching the whole file: the parent chart declares a
# dozen dependencies, each with its own `version:` key, so any of them would
# satisfy a file-wide match while the meshery-operator entry sat untouched.
assert_stamped() {
  local file="$1" description="$2" pattern="$3"
  local haystack
  haystack="${4-$(cat "$file")}"
  if ! printf '%s\n' "$haystack" | grep -Eq -- "$pattern"; then
    echo "error: $file: $description was not stamped to $VERSION" >&2
    echo "       the stamp pattern no longer matches this file; fix hack/sync-downstream.sh" >&2
    exit 1
  fi
}

# declared_dependencies prints the name of every entry in a chart's dependencies
# block, one per line. `- name:` entries also appear under `maintainers:` at the
# same indentation, so the scan is scoped to the dependencies block and ends at
# the next top-level key.
declared_dependencies() {
  awk '
    /^dependencies:/ { in_deps = 1; next }
    in_deps && /^[^[:space:]#]/ { in_deps = 0 }
    in_deps && $1 == "-" && $2 == "name:" { print $3 }
  ' "$1"
}

# dependency_version prints the version one named entry of a chart's
# dependencies block records, for the assertion on the parent chart's bump.
# Same block scoping, same reason, as declared_dependencies.
dependency_version() {
  awk -v want="$2" '
    /^dependencies:/ { in_deps = 1; next }
    in_deps && /^[^[:space:]#]/ { in_deps = 0 }
    in_deps && $1 == "-" && $2 == "name:" { current = $3; next }
    in_deps && current == want && $1 == "version:" { print $2; exit }
  ' "$1"
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

# stamp_badge rewrites one helm-docs shields.io badge in a chart README, selected
# by its label. `AppVersion` is the operator release on the parent and on every
# subchart; `Version` is the chart's own version, so only the parent's is stamped
# and a subchart's is deliberately left alone (it tracks the subchart lifecycle).
# One helper for both because the two rewrites differ only in that label, and a
# second copy of this regex is a second place for its encoding rules to rot.
#
# Two things the pattern has to get right:
#   - It anchors on the leading `![` so the `Version` pattern cannot also match
#     the `AppVersion` badge, whose label ends with it.
#   - It consumes the whole message segment through the trailing `-informational`
#     rather than stopping at the first `-`. Stopping early left a prerelease
#     appending its suffix on every pass (`Version-1.0.6-rc.1-rc.1-`) instead of
#     replacing it, and left a stale `-rc.1` behind when the final release was
#     stamped over it.
#
# A chart with no README is not an error; a README whose badge stopped matching
# is, which is what the assertion covers - pinned through `-informational` so a
# mangled message segment cannot satisfy it on the version prefix alone.
stamp_badge() {
  local readme="$1" label="$2"
  [ -f "$readme" ] || return 0
  LABEL="$label" VERSION="$VERSION" BADGE_VERSION="$BADGE_VERSION" perl -pi -e '
    my ($l, $v, $b) = @ENV{qw(LABEL VERSION BADGE_VERSION)};
    s{!\[\Q$l\E: [^\]]*\]\(https://img\.shields\.io/badge/\Q$l\E-[^)]*?-informational}
     {![$l: $v](https://img.shields.io/badge/$l-$b-informational}g;
  ' "$readme"
  assert_stamped "$readme" "$label badge" \
    "!\[${label}: ${VERSION_RE}\]\(https://img\.shields\.io/badge/${label}-${BADGE_VERSION_RE}-informational"
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
#
#     Discovery and the operator chart's declared dependencies are cross-checked
#     in BOTH directions, because each direction is a different way the chart
#     ends up lying about what it installs:
#       - declared but not discovered: a dependency vendored as a .tgz rather
#         than unpacked is never reached by the walk and keeps whatever
#         appVersion it already had. That is the original drift.
#       - discovered but not declared: were meshery/meshery to vendor a
#         third-party chart under charts/ - an upstream NATS chart, mirroring
#         what this repo vendors under pkg/broker/chart - the walk would stamp
#         the operator release onto a chart that installs something else. Passing
#         over it quietly is no better, because silence is what let the first
#         direction rot for releases on end.
#     So both fail the sync rather than either being resolved by guesswork.
#
# A space-delimited string rather than an array: `${arr[*]}` on an EMPTY array
# is an unbound-variable error under `set -u` in bash 3.2, which ships as
# /bin/bash on macOS - so the zero-subchart case would abort with a confusing
# error instead of the dependency message below, which is the one that explains
# what actually went wrong.
DECLARED_SUBCHARTS=" "
while read -r dep; do
  [ -n "$dep" ] || continue
  DECLARED_SUBCHARTS+="$dep "
done < <(declared_dependencies "$OPERATOR_CHART/Chart.yaml")

STAMPED_SUBCHARTS=" "
for subchart_yaml in "$OPERATOR_CHART"/charts/*/Chart.yaml; do
  [ -f "$subchart_yaml" ] || continue
  subchart="$(basename "$(dirname "$subchart_yaml")")"
  case "$DECLARED_SUBCHARTS" in
    *" $subchart "*) ;;
    *) echo "error: $OPERATOR_CHART/charts/$subchart is not a declared dependency of the operator chart" >&2
       echo "       every subchart under charts/ is stamped with the operator release, so a chart that" >&2
       echo "       does not track that release must not sit there: declare it in the operator chart's" >&2
       echo "       dependencies if it is operator-owned, or remove the vendored copy if it is not" >&2
       exit 1 ;;
  esac
  stamp_app_version "$subchart_yaml" "subchart appVersion"
  stamp_badge "$(dirname "$subchart_yaml")/README.md" AppVersion
  STAMPED_SUBCHARTS+="$subchart "
done

while read -r dep; do
  [ -n "$dep" ] || continue
  case "$STAMPED_SUBCHARTS" in
    *" $dep "*) ;;
    *) echo "error: $OPERATOR_CHART declares dependency '$dep' but charts/$dep/Chart.yaml was not stamped" >&2
       echo "       only an unpacked subchart directory can be stamped; a vendored .tgz cannot be" >&2
       exit 1 ;;
  esac
done < <(declared_dependencies "$OPERATOR_CHART/Chart.yaml")

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
stamp_badge "$README" Version
stamp_badge "$README" AppVersion
VERSION="$VERSION" perl -pi -e '
  my $v = $ENV{VERSION};
  s{^\| image\.tag \| string \| `"[^"]*"`}{| image.tag | string | `"$v"`};
' "$README"
assert_stamped "$README" "image.tag row" \
  "^\| image\.tag \| string \| \`\"${VERSION_RE}\"\`"

# 3. Parent chart dependency bump + re-vendor. The rewrite is bounded to the
#    meshery-operator entry: it starts at that entry's name key and may cross
#    only further indented lines of the same list item, so an entry that has lost
#    its `version:` key cannot hand the rewrite to a neighbour's. The bump is
#    then asserted against the value read back out of THAT entry - the one
#    substitution in this script that a file-wide match could not check, and the
#    one whose silent no-op is invisible: with a non-exact constraint helm
#    resolves it happily and vendors the PREVIOUS operator archive.
#    helm dependency update leaves the repository-less adapter deps alone
#    ("Assuming it exists in the charts directory") and repackages only the
#    file://-sourced operator chart + Chart.lock.
#
#    The trigger is CONTENT, not version. `helm package` folds in the CRD
#    bundles, the subchart Chart.yamls and every README this script rewrites, so
#    a re-sync of a tag that is already vendored (the workflow's supported
#    workflow_dispatch path) would otherwise leave the archive carrying pre-stamp
#    bytes beside freshly stamped sources. meshery's appVersion check reads the
#    sources and would pass, so the drift this script exists to prevent would
#    survive silently inside the artifact users actually install from. A genuine
#    no-op re-run still changes nothing and still skips, so the lock timestamp
#    and tgz bytes do not churn.
perl -0pi -e "s/(name: meshery-operator\n(?:(?=[ \t])(?![ \t]*-)[^\n]*\n)*?[ \t]*version: )[^\n]*/\${1}$VERSION/" "$PARENT_CHART/Chart.yaml"
assert_stamped "$PARENT_CHART/Chart.yaml" "meshery-operator dependency version" \
  "^${VERSION_RE}\$" "$(dependency_version "$PARENT_CHART/Chart.yaml" meshery-operator)"
if [ "$(chart_checksum)" != "$CHART_BEFORE" ] \
   || [ ! -f "$PARENT_CHART/charts/meshery-operator-$VERSION.tgz" ] \
   || ! grep -A2 'name: meshery-operator' "$PARENT_CHART/Chart.lock" 2>/dev/null | grep -Eq "version: ${VERSION_RE}$"; then
  rm -f "$PARENT_CHART"/charts/meshery-operator-*.tgz
  "$HELM_BIN" dependency update "$PARENT_CHART" >/dev/null
fi

# 4. Retire the legacy duplicate CRD copy in the parent chart.
rm -f "$PARENT_CHART/crds/crds.yaml"
rmdir "$PARENT_CHART/crds" 2>/dev/null || true

if git -C "$MESHERY_DIR" status --porcelain -- install/kubernetes/helm | grep -q .; then
  echo "sync: updated meshery charts to operator v$VERSION"
else
  echo "sync: no changes (already at v$VERSION)"
fi
