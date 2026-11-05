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

func TestBuildImage(t *testing.T) {
	tests := []struct {
		name        string
		customRepo  string
		version     string
		defaultRepo string
		defaultTag  string
		wantImage   string
		wantPolicy  corev1.PullPolicy
	}{
		{
			name:        "MeshSync default",
			customRepo:  "",
			version:     "",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "meshery/meshsync:1.0.3",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "MeshSync custom repo with default version",
			customRepo:  "myregistry.corp.internal/meshery/meshsync",
			version:     "",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "myregistry.corp.internal/meshery/meshsync:1.0.3",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "MeshSync custom repo with explicit version",
			customRepo:  "myregistry.corp.internal/meshery/meshsync",
			version:     "v1.0.4",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "myregistry.corp.internal/meshery/meshsync:1.0.4",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Private registry host with port and no tag",
			customRepo:  "localhost:5000/meshery/meshsync",
			version:     "1.0.4",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "localhost:5000/meshery/meshsync:1.0.4",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Private registry host with port and default version",
			customRepo:  "localhost:5000/meshery/meshsync",
			version:     "",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "localhost:5000/meshery/meshsync:1.0.3",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Custom repo contains tag directly with empty version",
			customRepo:  "myregistry.corp.internal/meshery/meshsync:1.0.5",
			version:     "",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "myregistry.corp.internal/meshery/meshsync:1.0.5",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Custom repo contains v-prefixed tag directly with empty version",
			customRepo:  "myregistry.corp.internal/meshery/meshsync:v1.0.5",
			version:     "",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "myregistry.corp.internal/meshery/meshsync:v1.0.5",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Custom repo contains tag but explicit version overrides it",
			customRepo:  "myregistry.corp.internal/meshery/meshsync:old-tag",
			version:     "v1.0.6",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "myregistry.corp.internal/meshery/meshsync:1.0.6",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Custom repo with sha256 digest",
			customRepo:  "myregistry.corp.internal/meshery/meshsync@sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
			version:     "",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "myregistry.corp.internal/meshery/meshsync@sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Custom repo with sha256 digest and explicit version (strips digest)",
			customRepo:  "myregistry.corp.internal/meshery/meshsync@sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
			version:     "1.0.5",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "myregistry.corp.internal/meshery/meshsync:1.0.5",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Custom repo with tag plus digest and explicit version (strips both digest and tag)",
			customRepo:  "myregistry.corp.internal/meshery/meshsync:v1.0@sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
			version:     "1.0.5",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "myregistry.corp.internal/meshery/meshsync:1.0.5",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Private registry host with port and tag plus digest with explicit version",
			customRepo:  "localhost:5000/meshery/meshsync:v1.0@sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
			version:     "1.0.5",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "localhost:5000/meshery/meshsync:1.0.5",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Broker NATS custom repo with default chart tag",
			customRepo:  "harbor.internal/nats",
			version:     "",
			defaultRepo: "nats",
			defaultTag:  "2.14.2-alpine",
			wantImage:   "harbor.internal/nats:2.14.2-alpine",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Broker NATS custom repo with explicit version",
			customRepo:  "harbor.internal/nats",
			version:     "2.10.0-alpine",
			defaultRepo: "nats",
			defaultTag:  "2.14.2-alpine",
			wantImage:   "harbor.internal/nats:2.10.0-alpine",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Empty defaultTag falls back to bare repository without trailing colon",
			customRepo:  "harbor.internal/nats",
			version:     "",
			defaultRepo: "nats",
			defaultTag:  "",
			wantImage:   "harbor.internal/nats",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Empty defaultRepo and empty defaultTag with custom repo containing tag",
			customRepo:  "harbor.internal/nats:2.14.2-alpine",
			version:     "",
			defaultRepo: "nats",
			defaultTag:  "",
			wantImage:   "harbor.internal/nats:2.14.2-alpine",
			wantPolicy:  corev1.PullIfNotPresent,
		},
		{
			name:        "Moving tag gets PullAlways",
			customRepo:  "",
			version:     "edge-latest",
			defaultRepo: "meshery/meshsync",
			defaultTag:  "1.0.3",
			wantImage:   "meshery/meshsync:edge-latest",
			wantPolicy:  corev1.PullAlways,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotImage, gotPolicy := BuildImage(tt.customRepo, tt.version, tt.defaultRepo, tt.defaultTag)
			if gotImage != tt.wantImage {
				t.Errorf("BuildImage() image = %q, want %q", gotImage, tt.wantImage)
			}
			if gotPolicy != tt.wantPolicy {
				t.Errorf("BuildImage() policy = %v, want %v", gotPolicy, tt.wantPolicy)
			}
		})
	}
}
