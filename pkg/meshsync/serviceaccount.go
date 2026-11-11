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

package meshsync

import (
	"strings"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	// ServiceAccountEnv is the operator Deployment environment variable that
	// names the ServiceAccount MeshSync pods run under. It is a cross-repo
	// contract: config/manager sets it to "meshery-meshsync", the ServiceAccount
	// that config/manager/meshsync_service_account.yaml defines and
	// config/rbac/meshsync_workload_role*.yaml binds to the read-only
	// "meshery-meshsync-role", and the meshery-operator Helm chart in
	// meshery/meshery is expected to create the same three objects and set the
	// same value (pending chart change; until then the operator keeps the
	// legacy default).
	ServiceAccountEnv = "MESHSYNC_SERVICE_ACCOUNT"

	// LegacyServiceAccountName is what MeshSync runs under when
	// ServiceAccountEnv is unset: the operator's own ServiceAccount, which every
	// install already has. Charts that predate the dedicated ServiceAccount do
	// not create one, so defaulting to the dedicated name would leave MeshSync
	// pods failing to create ("serviceaccount not found") until the chart
	// caught up.
	//
	// Since the operator went least-privilege (#782) this account has no
	// cluster-wide read, so MeshSync running under it is forbidden from listing
	// most of what it watches. Setting ServiceAccountEnv is the fix; this
	// default only keeps older charts from regressing further.
	LegacyServiceAccountName = "meshery-operator"
)

// ResolveServiceAccountName turns the ServiceAccountEnv value the operator read
// at startup into the ServiceAccount MeshSync pods run under. An empty value
// selects LegacyServiceAccountName. Any other value must be a valid
// ServiceAccount name (a DNS-1123 subdomain). An invalid value is a deployment
// misconfiguration: it is reported as a structured error rather than replaced,
// so the operator refuses to start instead of applying a Deployment the API
// server would reject on every reconcile.
func ResolveServiceAccountName(value string) (string, error) {
	if value == "" {
		return LegacyServiceAccountName, nil
	}
	if problems := validation.IsDNS1123Subdomain(value); len(problems) > 0 {
		return "", ErrInvalidServiceAccountName(value, strings.Join(problems, "; "))
	}
	return value, nil
}

// ServerOption customizes the MeshSync Deployment GetServerObject builds.
type ServerOption func(*v1.Deployment)

// WithServiceAccountName runs the MeshSync pods under the named ServiceAccount.
// An empty name keeps the template default, LegacyServiceAccountName. The
// operator passes the value ResolveServiceAccountName returned at startup.
func WithServiceAccountName(name string) ServerOption {
	return func(d *v1.Deployment) {
		if name != "" {
			d.Spec.Template.Spec.ServiceAccountName = name
		}
	}
}
