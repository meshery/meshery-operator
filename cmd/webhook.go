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
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// enableWebhooksEnv names the environment variable that controls whether the
// manager serves the v1alpha1 <-> v1alpha2 conversion webhooks.
const enableWebhooksEnv = "ENABLE_WEBHOOKS"

// webhookServerOptions is the ONE description of the webhook server. The
// manager is built from it and resolveWebhookEnablement checks the serving
// certificate against it, so the path the decision inspects is by construction
// the path the server would load.
//
// CertDir, CertName and KeyName are stated explicitly instead of left to
// controller-runtime's (unexported) defaulting so that both consumers read the
// same values. They equal controller-runtime's defaults, which is also where
// every manifest mounts the certificate (config/default/manager_webhook_patch.yaml
// and the meshery-operator Helm chart); TestWebhookServerOptionsMatchControllerRuntimeDefaults
// fails if a controller-runtime bump moves them.
func webhookServerOptions() webhook.Options {
	return webhook.Options{
		Port:     9443,
		CertDir:  filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs"),
		CertName: "tls.crt",
		KeyName:  "tls.key",
	}
}

// webhookEnablement is the outcome of resolveWebhookEnablement: whether to
// register the conversion webhooks, and why, for the startup log.
type webhookEnablement struct {
	Reason  string
	Enabled bool
}

// resolveWebhookEnablement decides whether the manager serves the conversion
// webhooks, from the ENABLE_WEBHOOKS value and the serving certificate files
// named by opts.
//
//   - "true" (any strconv.ParseBool true form): serve them. A missing
//     certificate is a misconfiguration and returns an error, so the process
//     exits with a structured message instead of crash-looping inside the
//     manager on "open .../tls.crt: no such file or directory".
//   - "false" (any ParseBool false form): do not serve them.
//   - unset or empty: serve them only when both the certificate and key exist.
//     When neither exists the manager runs without them (Enabled=false with a
//     Reason the caller logs as a warning). Operator charts older than v1.0.51
//     set no ENABLE_WEBHOOKS and mount no certificate, and they track the moving
//     stable-latest image, so defaulting ON crash-looped every one of them on
//     each new image (meshery/meshery#22555). Running without the webhooks is
//     safe while the CRDs those charts install use conversion strategy None.
//     A certificate without its key (or the reverse) is a broken mount, not an
//     absent one, and returns an error.
//   - anything else: an error, rather than guessing the operator's intent.
func resolveWebhookEnablement(enableWebhooks string, opts webhook.Options) (webhookEnablement, error) {
	value := strings.TrimSpace(enableWebhooks)

	certPath := filepath.Join(opts.CertDir, opts.CertName)
	keyPath := filepath.Join(opts.CertDir, opts.KeyName)

	if value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return webhookEnablement{}, ErrInvalidEnableWebhooks(value)
		}
		if !enabled {
			return webhookEnablement{Enabled: false, Reason: enableWebhooksEnv + " is false"}, nil
		}
		missing, err := missingServingCertFiles(certPath, keyPath)
		if err != nil {
			return webhookEnablement{}, err
		}
		if len(missing) > 0 {
			return webhookEnablement{}, ErrWebhookServingCertMissing(opts.CertDir, missing)
		}
		return webhookEnablement{Enabled: true, Reason: enableWebhooksEnv + " is true"}, nil
	}

	missing, err := missingServingCertFiles(certPath, keyPath)
	if err != nil {
		return webhookEnablement{}, err
	}
	switch len(missing) {
	case 0:
		return webhookEnablement{
			Enabled: true,
			Reason:  enableWebhooksEnv + " is unset and the serving certificate is present in " + opts.CertDir,
		}, nil
	case 2:
		return webhookEnablement{
			Enabled: false,
			Reason: enableWebhooksEnv + " is unset and no serving certificate is mounted at " + opts.CertDir +
				"; this is safe only while the CRDs use conversion strategy None. Set " + enableWebhooksEnv +
				"=true and mount a serving certificate to serve conversion webhooks",
		}, nil
	default:
		return webhookEnablement{}, ErrWebhookServingCertIncomplete(opts.CertDir, missing)
	}
}

// missingServingCertFiles returns which of the given files do not exist. Any
// other stat failure, or a path that is a directory, is returned as an error:
// it is a broken mount, not an absent one, and treating it as absent would
// silently turn the webhooks off.
func missingServingCertFiles(paths ...string) ([]string, error) {
	var missing []string
	for _, p := range paths {
		info, err := os.Stat(p)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			missing = append(missing, p)
		case err != nil:
			return nil, ErrWebhookServingCertUnreadable(p, err.Error())
		case info.IsDir():
			return nil, ErrWebhookServingCertUnreadable(p, "the path is a directory, not a file")
		}
	}
	return missing, nil
}
