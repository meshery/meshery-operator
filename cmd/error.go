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
	"strings"

	meshkiterrors "github.com/meshery/meshkit/errors"
)

// Error codes. Names and codes are unique within the meshery-operator component;
// allocate the next free code from helpers/component_info.json and bump its
// next_error_code.
const (
	ErrInvalidEnableWebhooksCode        = "1022"
	ErrWebhookServingCertMissingCode    = "1023"
	ErrWebhookServingCertIncompleteCode = "1024"
	ErrWebhookServingCertUnreadableCode = "1025"
)

// ErrInvalidEnableWebhooks is returned when ENABLE_WEBHOOKS holds a value that
// is neither a boolean nor empty.
func ErrInvalidEnableWebhooks(value string) error {
	return meshkiterrors.New(ErrInvalidEnableWebhooksCode, meshkiterrors.Fatal,
		[]string{"ENABLE_WEBHOOKS has an unrecognized value"},
		[]string{"ENABLE_WEBHOOKS=" + value + " is not a boolean"},
		[]string{"The operator Deployment sets ENABLE_WEBHOOKS to a value other than true, false, or empty"},
		[]string{"Set ENABLE_WEBHOOKS to \"true\" to serve the conversion webhooks, \"false\" to disable them, or leave it unset to serve them only when a serving certificate is mounted"})
}

// ErrWebhookServingCertMissing is returned when ENABLE_WEBHOOKS=true but the
// serving certificate or key is absent.
func ErrWebhookServingCertMissing(certDir string, missing []string) error {
	return meshkiterrors.New(ErrWebhookServingCertMissingCode, meshkiterrors.Fatal,
		[]string{"Conversion webhooks are enabled but the webhook serving certificate is missing"},
		[]string{"ENABLE_WEBHOOKS=true, but these files do not exist in " + certDir + ": " + strings.Join(missing, ", ")},
		[]string{"The webhook serving certificate Secret is not mounted into the manager container, or cert-manager has not issued it"},
		[]string{"Mount the serving certificate Secret (tls.crt and tls.key) at the webhook server's certificate directory, or set ENABLE_WEBHOOKS=false when the CRDs use conversion strategy None"})
}

// ErrWebhookServingCertIncomplete is returned when ENABLE_WEBHOOKS is unset and
// only one of the serving certificate and key is present.
func ErrWebhookServingCertIncomplete(certDir string, missing []string) error {
	return meshkiterrors.New(ErrWebhookServingCertIncompleteCode, meshkiterrors.Fatal,
		[]string{"The webhook serving certificate mount is incomplete"},
		[]string{"Only part of the serving certificate is present in " + certDir + "; missing: " + strings.Join(missing, ", ")},
		[]string{"The mounted Secret lacks tls.crt or tls.key, so the conversion webhook server cannot start"},
		[]string{"Mount a Secret carrying both tls.crt and tls.key, or set ENABLE_WEBHOOKS=false when the CRDs use conversion strategy None"})
}

// ErrWebhookServingCertUnreadable is returned when a serving certificate path
// exists but cannot be inspected or is not a file.
func ErrWebhookServingCertUnreadable(path, cause string) error {
	return meshkiterrors.New(ErrWebhookServingCertUnreadableCode, meshkiterrors.Fatal,
		[]string{"The webhook serving certificate cannot be read"},
		[]string{path + ": " + cause},
		[]string{"The certificate path is not a readable file: a permission problem, or a volume mounted over the file path"},
		[]string{"Mount the serving certificate Secret as a directory holding tls.crt and tls.key, readable by the manager's user"})
}
