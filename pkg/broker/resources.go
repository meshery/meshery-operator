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

package broker

// The NATS server topology is no longer hand-authored here. It is the official
// NATS Helm chart's rendered output, vendored in manifests/nats.gen.yaml,
// embedded (embed.go), and Server-Side-Applied by the controller. The operator
// overlays a few per-Broker fields onto those objects (see broker.go). Only the
// naming/label constants the operator still needs live here.
const (
	mesheryName       = "meshery"
	appLabelKey       = "app"
	componentLabelKey = "component"
	brokerComponent   = "broker"

	// natsServiceName is the release name used to render the vendored chart, and
	// therefore the fixed name of the NATS client Service, headless Service, and
	// StatefulSet. The operator reads the client Service / StatefulSet by this
	// name regardless of the Broker CR's name.
	natsServiceName = "meshery-nats"

	// clientPortName is the conventional NATS client port name; endpoint
	// derivation also recognises the chart's "nats" port name and port 4222.
	clientPortName = "client"

	// natsName is the official chart's canonical name for both the NATS server
	// container and the client port.
	natsName = "nats"

	// AuthSecretName holds the NATS token that the vendored StatefulSet reads via
	// the $NATS_TOKEN env var. The operator generates it at runtime; it is never
	// committed to source (replacing the old committed account JWT).
	AuthSecretName = "meshery-nats-auth" //nolint:gosec // G101: the Secret object's name, not a credential
)

var (
	// MesheryLabel / BrokerLabel are applied to operator-generated objects (e.g.
	// the auth Secret). Chart-rendered objects carry the chart's own labels.
	MesheryLabel = map[string]string{appLabelKey: mesheryName}
	BrokerLabel  = map[string]string{
		appLabelKey:       mesheryName,
		componentLabelKey: brokerComponent,
	}
)
