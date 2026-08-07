# shellcheck shell=bash
# The release-version guard shared by hack/sync-downstream.sh and
# hack/stamp-operator-version.sh.
#
# Both scripts stamp a version into artifacts users apply, on either side of the
# same release, so both must accept exactly the same set of versions. They used
# to carry a copy of the check each. That is a drift waiting to happen - and it
# had already started: the copies were identical only by luck, and a fix to one
# would not have reached the other. One implementation, sourced by both, removes
# the possibility rather than adding a test that the two copies still agree.
#
# Source it from a sibling script with:
#   . "$(dirname "${BASH_SOURCE[0]}")/lib/release-version.sh"

# RELEASE_VERSION_RE is the SemVer grammar for a release version, deliberately
# not a loose "digits, dots and a suffix" approximation.
#
# The loose form accepted 01.2.3, 1.2.3-01, 1.0.5-. and 1.0.5-rc..1 - none of
# which is a version. Those are worse than the moving tag this guard was written
# to stop: the stamp succeeds and writes the artifact, so the first sign of
# trouble is helm or an image pull failing opaquely on a version that never
# existed. Numeric identifiers therefore reject leading zeros, and every
# prerelease identifier must be non-empty.
#
# Build metadata (+...) is rejected on purpose even though SemVer allows it: the
# same value becomes a container image tag, and '+' is not legal in one.
RELEASE_VERSION_RE='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)'
RELEASE_VERSION_RE="$RELEASE_VERSION_RE"'(-(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)'
RELEASE_VERSION_RE="$RELEASE_VERSION_RE"'(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?$'

# require_release_version exits non-zero unless "$1" is a bare release version.
# "$2" is the consequence to name in the error - what would have been stamped
# with a bad value - so each caller's failure reads as its own.
require_release_version() {
  local version="$1" consequence="$2"

  case "$version" in
    v*) echo "error: version must be bare (1.2.3), got '$version'" >&2; return 1 ;;
  esac
  if ! printf '%s' "$version" | grep -Eq "$RELEASE_VERSION_RE"; then
    echo "error: '$version' is not a semantic version; $consequence" >&2
    return 1
  fi
}
