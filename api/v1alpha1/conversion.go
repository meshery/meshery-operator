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

package v1alpha1

import (
	"encoding/json"

	v1alpha2 "github.com/meshery/meshery-operator/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// v1alpha1 is a conversion spoke; v1alpha2 is the hub (storage) version. The two
// schemas are currently field-identical, so conversion is a lossless JSON
// round-trip of the spec and status (ObjectMeta is copied directly). When
// v1alpha2 later diverges, replace the round-trip with explicit field mapping.

// convertJSON copies in -> out via JSON, which is lossless while the two
// versions share identical JSON tags.
func convertJSON(in, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// ConvertTo converts this v1alpha1 Broker to the hub version (v1alpha2).
func (b *Broker) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha2.Broker)
	dst.ObjectMeta = b.ObjectMeta
	if err := convertJSON(b.Spec, &dst.Spec); err != nil {
		return err
	}
	return convertJSON(b.Status, &dst.Status)
}

// ConvertFrom converts from the hub version (v1alpha2) to this v1alpha1 Broker.
func (b *Broker) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha2.Broker)
	b.ObjectMeta = src.ObjectMeta
	if err := convertJSON(src.Spec, &b.Spec); err != nil {
		return err
	}
	return convertJSON(src.Status, &b.Status)
}

// ConvertTo converts this v1alpha1 MeshSync to the hub version (v1alpha2).
func (m *MeshSync) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha2.MeshSync)
	dst.ObjectMeta = m.ObjectMeta
	if err := convertJSON(m.Spec, &dst.Spec); err != nil {
		return err
	}
	return convertJSON(m.Status, &dst.Status)
}

// ConvertFrom converts from the hub version (v1alpha2) to this v1alpha1 MeshSync.
func (m *MeshSync) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha2.MeshSync)
	m.ObjectMeta = src.ObjectMeta
	if err := convertJSON(src.Spec, &m.Spec); err != nil {
		return err
	}
	return convertJSON(src.Status, &m.Status)
}
