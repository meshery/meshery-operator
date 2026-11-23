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

package main

import (
	"flag"
	"testing"
)

const (
	enableLeaderElectionFlag = "--enable-leader-election"
	testCustomNamespace      = "custom-ns"
	testOLMNamespace         = "openshift-operators"
)

func TestResolveLeaderElectionNamespace(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		flagNamespace string
		envNamespace  string
		want          string
	}{
		{
			name: "empty flag and env leave namespace empty for in-cluster detection",
			want: "",
		},
		{
			name:          "whitespace-only flag and env leave namespace empty",
			flagNamespace: "   ",
			envNamespace:  "\t\n",
			want:          "",
		},
		{
			name:         "POD_NAMESPACE is used when flag is unset",
			envNamespace: testOLMNamespace,
			want:         testOLMNamespace,
		},
		{
			name:          "POD_NAMESPACE is used when flag is whitespace",
			flagNamespace: "  ",
			envNamespace:  " operators ",
			want:          "operators",
		},
		{
			name:          "explicit flag overrides POD_NAMESPACE",
			flagNamespace: testCustomNamespace,
			envNamespace:  "operators",
			want:          testCustomNamespace,
		},
		{
			name:          "explicit flag trims surrounding whitespace",
			flagNamespace: " meshery-system ",
			want:          "meshery-system",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveLeaderElectionNamespace(tc.flagNamespace, tc.envNamespace)
			if got != tc.want {
				t.Fatalf("resolveLeaderElectionNamespace(%q, %q) = %q, want %q", tc.flagNamespace, tc.envNamespace, got, tc.want)
			}
		})
	}
}

func TestBindManagerFlagsAndOptions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		envNamespace   string
		wantNamespace  string
		args           []string
		wantLeaderElec bool
	}{
		{
			name:           "manifest defaults do not pin meshery namespace",
			args:           []string{enableLeaderElectionFlag, "--metrics-addr=:8443", "--health-probe-bind-address=:8081"},
			wantNamespace:  "",
			wantLeaderElec: true,
		},
		{
			name:           "POD_NAMESPACE applies when --namespace is omitted",
			args:           []string{enableLeaderElectionFlag},
			envNamespace:   testOLMNamespace,
			wantNamespace:  testOLMNamespace,
			wantLeaderElec: true,
		},
		{
			name:           "explicit --namespace flag overrides POD_NAMESPACE",
			args:           []string{enableLeaderElectionFlag, "--namespace=" + testCustomNamespace},
			envNamespace:   testOLMNamespace,
			wantNamespace:  testCustomNamespace,
			wantLeaderElec: true,
		},
		{
			name:           "leader election disabled by default",
			args:           nil,
			wantNamespace:  "",
			wantLeaderElec: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cfg managerConfig
			fs := flag.NewFlagSet("manager", flag.ContinueOnError)
			bindManagerFlags(fs, &cfg)
			if err := fs.Parse(tc.args); err != nil {
				t.Fatalf("fs.Parse(%v) returned unexpected error: %v", tc.args, err)
			}

			ns := resolveLeaderElectionNamespace(cfg.namespace, tc.envNamespace)
			opts := newManagerOptions(cfg, ns)

			if opts.LeaderElection != tc.wantLeaderElec {
				t.Errorf("LeaderElection = %v, want %v", opts.LeaderElection, tc.wantLeaderElec)
			}
			if opts.LeaderElectionID != leaderElectionID {
				t.Errorf("LeaderElectionID = %q, want %q", opts.LeaderElectionID, leaderElectionID)
			}
			if opts.LeaderElectionNamespace != tc.wantNamespace {
				t.Errorf("LeaderElectionNamespace = %q, want %q", opts.LeaderElectionNamespace, tc.wantNamespace)
			}
			if opts.Scheme != scheme {
				t.Errorf("Scheme was not wired to package scheme")
			}
			if !opts.Metrics.SecureServing {
				t.Errorf("Metrics.SecureServing = false, want true")
			}
			if opts.Metrics.BindAddress != ":8443" {
				t.Errorf("Metrics.BindAddress = %q, want %q", opts.Metrics.BindAddress, ":8443")
			}
			if opts.HealthProbeBindAddress != ":8081" {
				t.Errorf("HealthProbeBindAddress = %q, want %q", opts.HealthProbeBindAddress, ":8081")
			}
		})
	}
}
