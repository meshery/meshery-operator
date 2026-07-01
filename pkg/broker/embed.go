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

import (
	"bufio"
	"bytes"
	_ "embed"
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

// natsManifestsYAML is the official NATS Helm chart's rendered server manifests,
// vendored via `make nats-manifests`. The operator embeds and Server-Side-Applies
// them; Helm is never a runtime dependency (WS-6). Regenerate after editing
// pkg/broker/chart/values.yaml or bumping NATS_CHART_VERSION.
//
//go:embed manifests/nats.gen.yaml
var natsManifestsYAML []byte

// natsTemplate is the decoded set of chart objects, parsed once. Callers must
// deep-copy before mutating (see GetObjects).
var natsTemplate = mustDecodeNATSManifests()

func mustDecodeNATSManifests() []Object {
	objs, err := decodeNATSManifests(natsManifestsYAML)
	if err != nil {
		// The manifests are compile-time embedded and known-good; a decode
		// failure is a build defect, surfaced immediately by any test that
		// imports this package.
		panic("broker: cannot decode embedded NATS manifests: " + err.Error())
	}
	return objs
}

// decodeNATSManifests splits a multi-document YAML stream and decodes each
// document into a typed Kubernetes object using the client-go scheme.
func decodeNATSManifests(data []byte) ([]Object, error) {
	var out []Object
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		doc, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if isCommentOrBlank(doc) {
			continue
		}
		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(doc, nil, nil)
		if err != nil {
			return nil, err
		}
		if o, ok := obj.(Object); ok {
			out = append(out, o)
		}
	}
	return out, nil
}

// isCommentOrBlank reports whether a YAML document carries no object content
// (only blank lines and # comments), e.g. the generated-file header.
func isCommentOrBlank(doc []byte) bool {
	for _, line := range bytes.Split(doc, []byte("\n")) {
		t := bytes.TrimSpace(line)
		if len(t) == 0 || bytes.HasPrefix(t, []byte("#")) {
			continue
		}
		return false
	}
	return true
}
