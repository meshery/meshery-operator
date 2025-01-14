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
	"errors"
)

const (
	ErrGettingResourceCode  = "1013"
	ErrReplicasNotReadyCode = "1014"
	ErrConditionFalseCode   = "1015"
	ErrGettingEndpointCode  = "1016"
)

func ErrGettingResource(err error) error {
	return errors.New(ErrGettingResourceCode + ":" + "Unable to get resource" + err.Error())
}

func ErrGettingEndpoint(err error) error {
	return errors.New(ErrGettingEndpointCode + ":" + "Unable to get endpoint" + err.Error())
}

func ErrReplicasNotReady(reason string) error {
	return errors.New(ErrReplicasNotReadyCode + ":" + "The replicas are not ready" + reason)
}

func ErrConditionFalse(reason string) error {
	return errors.New(ErrConditionFalseCode + ":" + "The condition is false" + reason)
}
