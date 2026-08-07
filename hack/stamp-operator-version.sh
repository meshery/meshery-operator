#!/usr/bin/env bash
# Stamp a published operator release version into every in-repo artifact that
# names the manager image.
#
# Usage: hack/stamp-operator-version.sh <version>
#   <version>  Bare operator release version, e.g. 1.0.4 (no leading v).
#              It must already exist as meshery/meshery-operator:<version> on
#              Docker Hub - these artifacts are what users apply, so they may
#              only ever name an image that resolves.
#
# Why this exists: the manager image used to be pinned to the moving
# `stable-latest` tag, so a manifest or chart rendered months ago silently
# started pulling whatever master had become - which is how an install broke on
# a manager build that required webhook serving certs the old chart never
# mounted. Every artifact below now names an immutable release, and this script
# is the mechanism that moves them forward together instead of leaving three
# hand-edited copies to rot apart.
#
# What it stamps (idempotent - a second run produces no further changes):
#   1. Makefile          OPERATOR_RELEASE_VERSION (the single source of truth
#                        that IMG, and therefore `make bundle`, derives from).
#   2. config/manager/   manager.yaml image + kustomization.yaml newTag.
#   3. config/manifests/default.yaml - re-rendered from config/default by
#                        `make operator-manifest` (skip with SKIP_RENDER=1;
#                        the CI drift gate re-renders it either way).
#   4. bundle/manifests/*.clusterserviceversion.yaml - the manager image inside
#                        the frozen OLM bundle. `make bundle` would rewrite this
#                        from IMG, but that needs the operator-sdk CLI, which is
#                        not in CI; the stamp is byte-identical to what
#                        operator-sdk writes for the image field alone.
#
# Called by .github/workflows/stamp-release.yml on every published release, so
# master always names the newest release that actually exists. Run it by hand
# with `make stamp-release VERSION=<version>`.
set -euo pipefail

VERSION="${1:?usage: hack/stamp-operator-version.sh <version>}"
case "$VERSION" in
  v*) echo "error: version must be bare (1.2.3), got '$VERSION'" >&2; exit 1 ;;
esac
if ! printf '%s' "$VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
  echo "error: '$VERSION' is not a semantic version; the manager image must never be pinned to a moving tag" >&2
  exit 1
fi

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_DIR"

IMAGE_REPO="meshery/meshery-operator"
CSV="bundle/manifests/meshery-operator.clusterserviceversion.yaml"
STAMPED=(Makefile config/manager/manager.yaml config/manager/kustomization.yaml \
         config/manifests/default.yaml "$CSV")

# Compare the stamped files against their own prior content, not against git
# HEAD: run from a tree that already carries unrelated edits (a release PR, a
# rebase), a HEAD comparison would always report a change and hide a no-op.
checksum() { cat "${STAMPED[@]}" 2>/dev/null | shasum; }
BEFORE="$(checksum)"

# 1. Makefile single source of truth.
perl -pi -e "s{^OPERATOR_RELEASE_VERSION \?= .*}{OPERATOR_RELEASE_VERSION ?= $VERSION}" Makefile

# 2. The kustomize manager base: the literal image and the transformer's newTag
#    (kustomize rewrites nothing unless the transformer entry matches the image
#    name exactly as written in manager.yaml).
perl -pi -e "s{image: \Q$IMAGE_REPO\E:.*}{image: $IMAGE_REPO:$VERSION}" config/manager/manager.yaml
perl -pi -e "s{^  newTag: .*}{  newTag: $VERSION}" config/manager/kustomization.yaml

# 4. The frozen OLM bundle (before the render, so a failed render leaves no
#    half-stamped tree behind that a re-run would not fix).
[ -f "$CSV" ] && perl -pi -e "s{image: \Q$IMAGE_REPO\E:.*}{image: $IMAGE_REPO:$VERSION}" "$CSV"

# 3. The flat install manifest mesheryctl fetches, re-rendered from config/.
if [ "${SKIP_RENDER:-}" != "1" ]; then
  make operator-manifest
fi

if [ "$(checksum)" = "$BEFORE" ]; then
  echo "stamp: no changes (already at $VERSION)"
else
  echo "stamp: operator artifacts now pin $IMAGE_REPO:$VERSION"
fi
