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
	"net/http"
	"os"
	"path/filepath"
	"testing"

	meshkiterrors "github.com/meshery/meshkit/errors"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// certState is what a test materialises in the webhook certificate directory.
type certState string

const (
	certNone      certState = "no cert, no key"
	certBoth      certState = "cert and key"
	certOnly      certState = "cert without key"
	keyOnly       certState = "key without cert"
	certPathIsDir certState = "cert path is a directory"
)

// optionsWithCerts returns the production webhook options re-pointed at a fresh
// temporary directory populated per state. Only CertDir is overridden, so the
// file names under test are the ones the manager uses.
func optionsWithCerts(t *testing.T, state certState) webhook.Options {
	t.Helper()
	opts := webhookServerOptions()
	opts.CertDir = t.TempDir()

	write := func(name string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(opts.CertDir, name), []byte("test"), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	switch state {
	case certNone:
	case certBoth:
		write(opts.CertName)
		write(opts.KeyName)
	case certOnly:
		write(opts.CertName)
	case keyOnly:
		write(opts.KeyName)
	case certPathIsDir:
		if err := os.Mkdir(filepath.Join(opts.CertDir, opts.CertName), 0o700); err != nil {
			t.Fatalf("creating directory at cert path: %v", err)
		}
		write(opts.KeyName)
	default:
		t.Fatalf("unknown cert state %q", state)
	}
	return opts
}

func TestResolveWebhookEnablement(t *testing.T) {
	allStates := []certState{certNone, certBoth, certOnly, keyOnly, certPathIsDir}

	// want describes one outcome: either an enablement or a MeshKit error code.
	type want struct {
		errCode string
		enabled bool
	}
	enabled := want{enabled: true}
	disabled := want{enabled: false}
	fails := func(code string) want { return want{errCode: code} }

	cases := []struct {
		outcomes map[certState]want
		env      string
	}{
		// Unset: the old-chart case (meshery/meshery#22555). Serve only when the
		// certificate is fully mounted; run without webhooks when it is absent.
		{env: "", outcomes: map[certState]want{
			certNone:      disabled,
			certBoth:      enabled,
			certOnly:      fails(ErrWebhookServingCertIncompleteCode),
			keyOnly:       fails(ErrWebhookServingCertIncompleteCode),
			certPathIsDir: fails(ErrWebhookServingCertUnreadableCode),
		}},
		// Whitespace-only is treated as unset, not as an invalid value.
		{env: "  ", outcomes: map[certState]want{
			certNone: disabled,
			certBoth: enabled,
		}},
		// Explicit true: a missing certificate is a misconfiguration.
		{env: "true", outcomes: map[certState]want{
			certNone:      fails(ErrWebhookServingCertMissingCode),
			certBoth:      enabled,
			certOnly:      fails(ErrWebhookServingCertMissingCode),
			keyOnly:       fails(ErrWebhookServingCertMissingCode),
			certPathIsDir: fails(ErrWebhookServingCertUnreadableCode),
		}},
		// The Helm chart quotes a bool; accept every ParseBool spelling.
		{env: "TRUE", outcomes: map[certState]want{
			certNone: fails(ErrWebhookServingCertMissingCode),
			certBoth: enabled,
		}},
		{env: " 1 ", outcomes: map[certState]want{
			certNone: fails(ErrWebhookServingCertMissingCode),
			certBoth: enabled,
		}},
		// Explicit false: never serve, and never inspect the certificate.
		{env: "false", outcomes: map[certState]want{
			certNone:      disabled,
			certBoth:      disabled,
			certOnly:      disabled,
			keyOnly:       disabled,
			certPathIsDir: disabled,
		}},
		{env: "False", outcomes: map[certState]want{
			certNone: disabled,
			certBoth: disabled,
		}},
		{env: "0", outcomes: map[certState]want{
			certBoth: disabled,
		}},
		// Anything else is refused rather than guessed at.
		{env: "yes", outcomes: map[certState]want{
			certNone: fails(ErrInvalidEnableWebhooksCode),
			certBoth: fails(ErrInvalidEnableWebhooksCode),
		}},
		{env: "enabled", outcomes: map[certState]want{
			certNone: fails(ErrInvalidEnableWebhooksCode),
			certBoth: fails(ErrInvalidEnableWebhooksCode),
		}},
	}

	// Unset, true and false are the contract's three modes: each must state an
	// outcome for every certificate state, so a new state cannot slip in untested.
	exhaustive := map[string]bool{"": true, "true": true, "false": true}

	for _, tc := range cases {
		for _, state := range allStates {
			w, ok := tc.outcomes[state]
			if !ok {
				if exhaustive[tc.env] {
					t.Errorf("ENABLE_WEBHOOKS=%q has no expected outcome for %q", tc.env, state)
				}
				continue
			}
			t.Run("ENABLE_WEBHOOKS="+tc.env+"/"+string(state), func(t *testing.T) {
				opts := optionsWithCerts(t, state)
				got, err := resolveWebhookEnablement(tc.env, opts)

				if w.errCode != "" {
					if err == nil {
						t.Fatalf("want error code %s, got enablement %+v", w.errCode, got)
					}
					if code := meshkiterrors.GetCode(err); code != w.errCode {
						t.Fatalf("want error code %s, got %s (%v)", w.errCode, code, err)
					}
					if got != (webhookEnablement{}) {
						t.Fatalf("want zero enablement alongside an error, got %+v", got)
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.Enabled != w.enabled {
					t.Fatalf("want Enabled=%t, got %+v", w.enabled, got)
				}
				if got.Reason == "" {
					t.Fatal("want a Reason for the startup log, got none")
				}
			})
		}
	}
}

// TestWebhookServerOptionsMatchControllerRuntimeDefaults pins the certificate
// location stated in webhookServerOptions to controller-runtime's own defaults,
// which is where every manifest mounts the serving certificate. Registering a
// handler is the public trigger for the server's defaulting.
func TestWebhookServerOptionsMatchControllerRuntimeDefaults(t *testing.T) {
	srv, ok := webhook.NewServer(webhook.Options{}).(*webhook.DefaultServer)
	if !ok {
		t.Fatal("webhook.NewServer no longer returns *webhook.DefaultServer; re-derive the default comparison")
	}
	srv.Register("/defaulting-probe", http.NotFoundHandler())

	ours := webhookServerOptions()
	defaults := srv.Options
	if ours.CertDir != defaults.CertDir || ours.CertName != defaults.CertName || ours.KeyName != defaults.KeyName {
		t.Fatalf("webhookServerOptions certificate location %s/{%s,%s} differs from controller-runtime's default %s/{%s,%s}",
			ours.CertDir, ours.CertName, ours.KeyName, defaults.CertDir, defaults.CertName, defaults.KeyName)
	}
	if ours.Port != defaults.Port {
		t.Fatalf("webhookServerOptions port %d differs from controller-runtime's default %d", ours.Port, defaults.Port)
	}
}
