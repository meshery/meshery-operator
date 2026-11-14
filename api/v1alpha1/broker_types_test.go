/*
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

// Import go packages

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("The test cases for customize resource: Broker", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		BrokerName      string = "broker-test"
		BrokerNamespace string = "broker-test-namespace"
	)

	// create a Broker object
	broker := &Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BrokerName,
			Namespace: BrokerNamespace,
		},
		Spec: BrokerSpec{
			Size: 1,
		},
	}

	// use fake client to test the Broker object
	Context("When creating a Broker object", func() {

		ctx := context.Background()
		It("should create a Broker object successfully", func() {
			Expect(fakeClient.Create(ctx, broker)).Should(Succeed())
		})

		It("should get the Broker object successfully", func() {
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Name:      BrokerName,
				Namespace: BrokerNamespace,
			}, broker)).Should(Succeed())
		})

		It("should update the Broker object successfully", func() {
			broker.Spec.Size = 2
			Expect(fakeClient.Update(ctx, broker)).Should(Succeed())
		})

		// check broker.Spec.Size is 2
		It("should get the Broker object successfully", func() {
			broker2 := &Broker{}
			Expect(fakeClient.Get(ctx, client.ObjectKey{
				Name:      BrokerName,
				Namespace: BrokerNamespace,
			}, broker2)).Should(Succeed())
			Expect(broker2.Spec.Size).Should(Equal(int32(2)))
		})

		// check broker list
		It("should list the Broker object successfully", func() {
			brokerList := &BrokerList{}
			Expect(fakeClient.List(ctx, brokerList)).Should(Succeed())
			Expect(len(brokerList.Items)).Should(Equal(1))
		})

		It("should delete the Broker object successfully", func() {
			Expect(fakeClient.Delete(ctx, broker)).Should(Succeed())
		})
	})
})

// sourceRangesPatterns returns the generated items.pattern of
// spec.service.loadBalancerSourceRanges for every served Broker version.
func sourceRangesPatterns(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "config", "crd", "bases", "meshery.io_brokers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.NewYAMLOrJSONDecoder(f, 4096).Decode(crd); err != nil {
		t.Fatal(err)
	}
	patterns := map[string]string{}
	for _, version := range crd.Spec.Versions {
		if version.Schema == nil || version.Schema.OpenAPIV3Schema == nil {
			t.Fatalf("version %s has no schema", version.Name)
		}
		field := version.Schema.OpenAPIV3Schema.Properties["spec"].Properties["service"].Properties["loadBalancerSourceRanges"]
		if field.Items == nil || field.Items.Schema == nil || field.Items.Schema.Pattern == "" {
			t.Fatalf("version %s: loadBalancerSourceRanges items carry no pattern", version.Name)
		}
		patterns[version.Name] = field.Items.Schema.Pattern
	}
	return patterns
}

// TestLoadBalancerSourceRangesPattern checks the CIDR pattern the CRD enforces
// on spec.service.loadBalancerSourceRanges. The API server matches it with Go's
// regexp, unanchored, exactly as here. It must be identical in every version,
// accept every canonical IPv4/IPv6 CIDR, reject the malformed values that used
// to reach the Service apply (the reported one being an unrendered template
// placeholder), and never accept a string net.ParseCIDR rejects.
func TestLoadBalancerSourceRangesPattern(t *testing.T) {
	patterns := sourceRangesPatterns(t)
	if len(patterns) < 2 {
		t.Fatalf("expected the pattern in v1alpha1 and v1alpha2, got %v", patterns)
	}
	var pattern string
	for version, p := range patterns {
		if pattern != "" && p != pattern {
			t.Fatalf("version %s pattern differs from the others; keep the markers in api/v1alpha1 and api/v1alpha2 identical", version)
		}
		pattern = p
	}
	re := regexp.MustCompile(pattern)

	valid := []string{
		"203.0.113.0/24", "0.0.0.0/0", "10.0.0.1/32", "255.255.255.255/32", "192.168.0.0/16",
		"2001:db8::/32", "::/0", "::1/128", "fe80::1/64", "FE80::ABCD/10",
		"2001:0db8:0000:0000:0000:0000:0000:0001/128", "1:2:3:4:5:6:7::/112", "::2:3:4:5:6:7:8/128",
		"::ffff:192.0.2.1/128", "::192.0.2.1/128", "64:ff9b::192.0.2.33/96",
		"1:2:3:4:5:6:192.0.2.1/96", "1::2:192.0.2.1/96", "1:2::3:4:192.0.2.1/96",
	}
	for _, value := range valid {
		if _, _, err := net.ParseCIDR(value); err != nil {
			t.Fatalf("fixture %q is not a CIDR: %v", value, err)
		}
		if !re.MatchString(value) {
			t.Errorf("pattern rejects valid CIDR %q", value)
		}
	}

	invalid := []string{
		"${PLAYGROUND_EGRESS_CIDR}", "", "10.0.0.1", "10.0.0.0/", "10.0.0.0/33", "256.0.0.0/8",
		"10.0.0/8", "010.0.0.0/8", "10.0.0.0/08", "example.com/24", " 10.0.0.0/8", "10.0.0.0/8 ",
		"10.0.0.0/8,10.1.0.0/16", "2001:db8::/129", "2001:db8:::/32", "1:2:3:4:5:6:7:8:9/64",
		"12345::/16", "::ffff:999.0.0.1/128", "fe80::1%eth0/64", "2001:db8::", "1::2::3/64",
	}
	for _, value := range invalid {
		if re.MatchString(value) {
			t.Errorf("pattern accepts malformed value %q", value)
		}
	}

	// Property checks with a fixed seed: every canonical prefix matches, and
	// nothing the pattern matches is rejected by net.ParseCIDR.
	rng := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic test data, not security
	for range 20000 {
		value := randomPrefix(rng).String()
		if !re.MatchString(value) {
			t.Fatalf("pattern rejects canonical prefix %q", value)
		}
		mutated := mutate(rng, value)
		if re.MatchString(mutated) {
			if _, _, err := net.ParseCIDR(mutated); err != nil {
				t.Fatalf("pattern accepts %q, which net.ParseCIDR rejects: %v", mutated, err)
			}
		}
	}
}

// randomPrefix returns a random IPv4, IPv6 or IPv4-mapped IPv6 prefix with a
// random (not necessarily masked) address.
func randomPrefix(rng *rand.Rand) netip.Prefix {
	var raw [16]byte
	_, _ = rng.Read(raw[:])
	switch rng.Intn(4) {
	case 0:
		addr := netip.AddrFrom4([4]byte{raw[0], raw[1], raw[2], raw[3]})
		return netip.PrefixFrom(addr, rng.Intn(33))
	case 1:
		// Sparse groups exercise "::" compression.
		for i := range raw {
			if rng.Intn(3) == 0 {
				raw[i] = 0
			}
		}
		return netip.PrefixFrom(netip.AddrFrom16(raw), rng.Intn(129))
	case 2:
		v4 := netip.AddrFrom4([4]byte{raw[0], raw[1], raw[2], raw[3]})
		return netip.PrefixFrom(netip.AddrFrom16(v4.As16()), 96+rng.Intn(33))
	default:
		return netip.PrefixFrom(netip.AddrFrom16(raw), rng.Intn(129))
	}
}

// mutate changes, inserts or deletes one character of value.
func mutate(rng *rand.Rand, value string) string {
	const alphabet = "0123456789abcdefABCDEF:./ g%"
	b := []byte(value)
	i := rng.Intn(len(b))
	c := alphabet[rng.Intn(len(alphabet))]
	switch rng.Intn(3) {
	case 0:
		b[i] = c
		return string(b)
	case 1:
		return string(b[:i]) + string(c) + string(b[i:])
	default:
		return string(b[:i]) + string(b[i+1:])
	}
}

// externalEndpointOverridePatterns returns the generated pattern of
// spec.service.externalEndpointOverride for every served Broker version.
func externalEndpointOverridePatterns(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "config", "crd", "bases", "meshery.io_brokers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.NewYAMLOrJSONDecoder(f, 4096).Decode(crd); err != nil {
		t.Fatal(err)
	}
	patterns := map[string]string{}
	for _, version := range crd.Spec.Versions {
		if version.Schema == nil || version.Schema.OpenAPIV3Schema == nil {
			t.Fatalf("version %s has no schema", version.Name)
		}
		field := version.Schema.OpenAPIV3Schema.Properties["spec"].Properties["service"].Properties["externalEndpointOverride"]
		if field.Pattern == "" {
			t.Fatalf("version %s: externalEndpointOverride carries no pattern", version.Name)
		}
		patterns[version.Name] = field.Pattern
	}
	return patterns
}

// TestExternalEndpointOverridePattern checks the host:port pattern the CRD enforces
// on spec.service.externalEndpointOverride. The pattern must be identical across all served
// versions, accept canonical hostnames, FQDNs, IPv4, and bracketed IPv6 addresses with
// valid ports (1-65535), and reject scheme prefixes, paths, and malformed inputs.
func TestExternalEndpointOverridePattern(t *testing.T) {
	patterns := externalEndpointOverridePatterns(t)
	if len(patterns) < 2 {
		t.Fatalf("expected the pattern in v1alpha1 and v1alpha2, got %v", patterns)
	}
	var pattern string
	for version, p := range patterns {
		if pattern != "" && p != pattern {
			t.Fatalf("version %s pattern differs from the others; keep markers identical", version)
		}
		pattern = p
	}
	re := regexp.MustCompile(pattern)

	checkValidEndpoints(t, re)
	checkInvalidEndpoints(t, re)
	checkEndpointProperties(t, re)
}

func checkValidEndpoints(t *testing.T, re *regexp.Regexp) {
	t.Helper()
	valid := []string{
		"broker.example.com:4222", "broker.example.com.:4222", "localhost:4222", "localhost.:4222",
		"broker:4222", "broker:04222", "broker:0080", "meshery-nats:4222",
		"198.51.100.1:4222", "192.168.1.100:80", "203.0.113.195:65535", "my-broker.meshery.svc.cluster.local:1",
		"my-broker.meshery.svc.cluster.local.:1",
		"[2001:db8::1]:4222", "[::1]:4222", "[fe80::1]:4222", "[::ffff:192.0.2.1]:4222",
	}
	for _, value := range valid {
		host, portStr, err := net.SplitHostPort(value)
		if err != nil {
			t.Fatalf("fixture %q is not host:port: %v", value, err)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 || host == "" {
			t.Fatalf("fixture %q has invalid host or port", value)
		}
		if !re.MatchString(value) {
			t.Errorf("pattern rejects valid endpoint %q", value)
		}
	}
}

func checkInvalidEndpoints(t *testing.T, re *regexp.Regexp) {
	t.Helper()
	invalid := []string{
		"nats://broker.example.com:4222", "nats://198.51.100.1:4222", "tcp://localhost:4222", "http://broker:4222",
		"broker.example.com", "198.51.100.1", ":4222", "broker:0", "broker:00", "broker:65536", "broker:065536",
		"broker:70000", "broker.example.com:4222/", "broker.example.com:4222/path", "${BROKER_ENDPOINT}",
		"broker..example.com:4222", "-broker:4222", "broker-:4222", "2001:db8::1:4222", "[2001:db8::1]",
		"[2001:db8:::1]:4222", "[::ffff:999.0.0.1]:4222", "[2001:db8::1]:0", "[2001:db8::1]:0000",
		".:4222", " broker:4222", "broker:4222 ", "",
	}
	for _, value := range invalid {
		if re.MatchString(value) {
			t.Errorf("pattern accepts malformed endpoint %q", value)
		}
	}
}

func checkEndpointProperties(t *testing.T, re *regexp.Regexp) {
	t.Helper()
	rng := rand.New(rand.NewSource(2)) //nolint:gosec // deterministic test data, not security
	for range 10000 {
		value := randomValidEndpoint(rng)
		if !re.MatchString(value) {
			t.Fatalf("pattern rejects valid endpoint %q", value)
		}
		mutated := mutate(rng, value)
		if re.MatchString(mutated) {
			host, portStr, err := net.SplitHostPort(mutated)
			if err != nil {
				t.Fatalf("pattern accepts %q which net.SplitHostPort rejects: %v", mutated, err)
			}
			port, err := strconv.Atoi(portStr)
			if err != nil || port < 1 || port > 65535 || host == "" {
				t.Fatalf("pattern accepts %q with invalid port or empty host", mutated)
			}
			if strings.Contains(mutated, "[") && net.ParseIP(host) == nil {
				t.Fatalf("pattern accepts bracketed %q which net.ParseIP rejects", mutated)
			}
		}
	}
}

func randomValidEndpoint(rng *rand.Rand) string {
	port := 1 + rng.Intn(65535)
	switch rng.Intn(3) {
	case 0:
		labels := []string{"broker", "nats", "cluster", "local", "mesh", "test"}
		count := 1 + rng.Intn(3)
		name := ""
		for i := range count {
			if i > 0 {
				name += "."
			}
			name += labels[rng.Intn(len(labels))]
		}
		return fmt.Sprintf("%s:%d", name, port)
	case 1:
		return fmt.Sprintf("%d.%d.%d.%d:%d", 1+rng.Intn(223), rng.Intn(256), rng.Intn(256), 1+rng.Intn(254), port)
	default:
		var raw [16]byte
		_, _ = rng.Read(raw[:])
		addr := netip.AddrFrom16(raw)
		return fmt.Sprintf("[%s]:%d", addr.String(), port)
	}
}
