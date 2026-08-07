/*
Copyright Meshery Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	corev1 "k8s.io/api/core/v1"
)

// Image tag policy shared by every managed component (MeshSync's Deployment,
// the broker's NATS StatefulSet). It lives here rather than in each builder so
// the two cannot drift: a moving tag reached only one of them once, and the
// resulting silent-drift-on-pod-restart is exactly the failure this package
// exists to prevent.

// MovingTag reports whether tag is a channel tag - one the publisher
// re-points at a new image without changing the tag string ("latest",
// "stable-latest", "edge-latest").
//
// Moving tags are unfit as defaults: the image a cluster runs then changes
// silently on any pod restart, so a chart pinned months ago can suddenly pull
// a manager build that needs mounts the chart never provided. Pinned tags are
// immutable by convention, which is what makes IfNotPresent safe for them.
func MovingTag(tag string) bool {
	return tag == "latest" || strings.HasSuffix(tag, "-latest")
}

// PullPolicyFor returns the pull policy matching a tag's mutability: moving
// tags need PullAlways to track their channel, pinned tags get IfNotPresent so
// side-loaded images (kind, `docker save`) and air-gapped clusters work without
// reaching a registry.
func PullPolicyFor(tag string) corev1.PullPolicy {
	if MovingTag(tag) {
		return corev1.PullAlways
	}
	return corev1.PullIfNotPresent
}

// NormalizeImageTag drops a leading "v" from a tag that is otherwise a
// semantic version, because the registries backing the managed components
// publish bare-semver tags only: `meshery/meshsync:1.0.3` and
// `nats:2.14.2-alpine` exist, `meshery/meshsync:v1.0.3` 404s.
//
// The mismatch is a real trap, not a hypothetical: the git tags and GitHub
// release names for those same components ARE v-prefixed (v1.0.3), so a user
// pinning spec.version by copying the release they want gets ImagePullBackOff
// for a version that was published correctly. Tags that are not semantic
// versions ("stable-latest", "edge-v1.0.3", a commit sha) are returned
// unchanged - only a v followed by a parseable version is rewritten. The
// lenient parser is deliberate: partial pins ("v2.10") are as common as full
// ones and equally broken with the v.
func NormalizeImageTag(tag string) string {
	rest, found := strings.CutPrefix(tag, "v")
	if !found {
		return tag
	}
	if _, err := semver.NewVersion(rest); err != nil {
		return tag
	}
	return rest
}
