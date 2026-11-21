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
	meshkiterrors "github.com/meshery/meshkit/errors"
)

// Error codes. Names and codes are unique within the meshery-operator component;
// allocate the next free code from helpers/component_info.json and bump its
// next_error_code.
const (
	ErrInvalidConversionHubCode = "1023"
)

// ErrInvalidConversionHub is returned when a conversion receives a nil or
// unexpected hub object.
func ErrInvalidConversionHub(detail string) error {
	return meshkiterrors.New(ErrInvalidConversionHubCode, meshkiterrors.Alert,
		[]string{"Invalid conversion hub object"},
		[]string{detail},
		[]string{"The conversion webhook received a nil or unexpected hub object type"},
		[]string{"Check the conversion webhook configuration and the CRD versions in use"})
}
