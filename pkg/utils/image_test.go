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
	"testing"

	corev1 "k8s.io/api/core/v1"
)

// Representative tags from the registries the managed components pull from:
// the three moving channel tags, an immutable meshery release, an immutable
// nats release, and a commit-sha tag.
const (
	latest       = "latest"
	stableLatest = "stable-latest"
	edgeLatest   = "edge-latest"
	pinned       = "1.0.3"
	pinnedNats   = "2.14.2-alpine"
	shaTag       = "abc1234"
)

func TestMovingTag(t *testing.T) {
	cases := map[string]bool{
		latest:            true,
		stableLatest:      true,
		edgeLatest:        true,
		"nightly-latest":  true,
		pinned:            false,
		"v" + pinned:      false,
		pinnedNats:        false,
		"stable-v1.0.4":   false, // a channel-prefixed but immutable release tag
		shaTag:            false,
		"latest-stable":   false, // "latest" only counts as a suffix or the whole tag
		"1.0.3-latest.1":  false,
		"":                false,
		"edge-3e04e50":    false,
		"1.0.0-rc.1":      false,
		"some/repo:1.0.0": false,
	}
	for tag, want := range cases {
		if got := MovingTag(tag); got != want {
			t.Errorf("MovingTag(%q) = %v, want %v", tag, got, want)
		}
	}
}

func TestPullPolicyFor(t *testing.T) {
	cases := map[string]corev1.PullPolicy{
		stableLatest: corev1.PullAlways,
		edgeLatest:   corev1.PullAlways,
		latest:       corev1.PullAlways,
		pinned:       corev1.PullIfNotPresent,
		pinnedNats:   corev1.PullIfNotPresent,
		shaTag:       corev1.PullIfNotPresent,
	}
	for tag, want := range cases {
		if got := PullPolicyFor(tag); got != want {
			t.Errorf("PullPolicyFor(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestNormalizeImageTag(t *testing.T) {
	cases := map[string]string{
		"v" + pinned:  pinned,
		"v1.0.3-rc.1": "1.0.3-rc.1",
		"v2.10":       "2.10",
		pinned:        pinned,
		pinnedNats:    pinnedNats,
		// Not a v+version: left exactly as written.
		stableLatest:  stableLatest,
		"edge-v1.0.3": "edge-v1.0.3",
		latest:        latest,
		"vnext":       "vnext",
		"v":           "v",
		"":            "",
		shaTag:        shaTag,
	}
	for in, want := range cases {
		if got := NormalizeImageTag(in); got != want {
			t.Errorf("NormalizeImageTag(%q) = %q, want %q", in, got, want)
		}
	}
}
