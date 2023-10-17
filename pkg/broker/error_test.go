/*
Copyright 2023 Layer5, Inc.

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
	"testing"
)

func TestErrGettingResource(t *testing.T) {
	err := ErrGettingResource(errors.New("test"))
	if err == nil {
		t.Error("expected error but got nil")
	}
}

func TestErrGettingEndpoint(t *testing.T) {
	err := ErrGettingEndpoint(errors.New("test"))
	if err == nil {
		t.Error("expected error but got nil")
	}
}

func TestErrReplicasNotReady(t *testing.T) {
	err := ErrReplicasNotReady("test")
	if err == nil {
		t.Error("expected error but got nil")
	}
}

func TestErrConditionFalse(t *testing.T) {
	err := ErrConditionFalse("test")
	if err == nil {
		t.Error("expected error but got nil")
	}
}
