/*
Copyright 2020 Layer5, Inc.

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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	runtime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

// entry
func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "APIs Suite")
}

var fakeClient client.Client

var _ = BeforeSuite(func() {
	By("Initial the fake client for the test")

	// initial scheme
	scheme := runtime.NewScheme()
	// for normal resources
	_ = clientgoscheme.AddToScheme(scheme)
	// register customize resources
	err := AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	// initial fake client
	fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&MeshSync{}).Build()

})
