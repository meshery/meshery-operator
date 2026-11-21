package v1alpha1

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
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/*
Here we do not want test CURD of kubernetes service, we want to confirm that:
* our CRDs resouces group version could be support by specific kubernetes server
* the specific value of the CRDs field can be workwell without pruning by the api-server to etcd
*/
var _ = Describe("The test case for the meshsync CRDs", func() {

	ctx := context.Background()

	const (
		URL            string = "nats://meshery.io:4222"
		str            string = "healthy"
		Reason         string = "Testcase"
		Message        string = "Message for testcase"
		PublishingTo   string = "Publish for testcase"
		FileManager    string = "testcase-meshsync"
		createFirstMsg string = "Create the meshsync CRDs first"

		Kind             string = "MeshSync"
		APIVersion       string = "meshery.io/v1alpha1"
		defaultNamespace string = "default"
	)

	var meshSync *MeshSync
	var typeNamespace types.NamespacedName

	BeforeEach(func() {
		meshSync = &MeshSync{
			TypeMeta: metav1.TypeMeta{
				APIVersion: APIVersion,
				Kind:       Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: defaultNamespace,
				Name:      defaultNamespace,
			},
			Spec: MeshSyncSpec{
				Size: 2,
				Broker: MeshsyncBroker{
					Custom: CustomMeshsyncBroker{
						URL: URL,
					},
					Native: NativeMeshsyncBroker{
						Namespace: defaultNamespace,
						Name:      defaultNamespace,
					},
				},
				WatchList: corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1apha1",
						Kind:       "ConfigMap",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "watch-list",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"blacklist":  "",
						whitelistKey: "[{\"Resource\":\"namespaces.v1.\",\"Events\":[\"ADDED\",\"DELETE\"]},{\"Resource\":\"replicasets.v1.apps\",\"Events\":[\"ADDED\",\"DELETE\"]},{\"Resource\":\"pods.v1.\",\"Events\":[\"MODIFIED\"]}]",
					},
				},
			},
		}

		typeNamespace = types.NamespacedName{
			Namespace: defaultNamespace,
			Name:      defaultNamespace,
		}
	})

	AfterEach(func() {
		err := fakeClient.Delete(ctx, meshSync)
		if !apierrors.IsNotFound(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Context("The CURD case for the meshsync CRDs", func() {

		It("The meshsync CRDs create acticity should be succeed", func() {
			By("Create the meshsync CRDs")
			err := fakeClient.Create(ctx, meshSync)
			Expect(err).NotTo(HaveOccurred())
		})

		It("The meshsync CRDs get should be succeed", func() {
			By(createFirstMsg)
			err := fakeClient.Create(ctx, meshSync)
			Expect(err).NotTo(HaveOccurred())

			By("Get the meshsync CRDs")
			mesheSyncGet := &MeshSync{}
			err = fakeClient.Get(ctx, typeNamespace, mesheSyncGet)
			Expect(err).NotTo(HaveOccurred())

			By("Confirm the URL equal to nats://meshery.io:4222")
			url := mesheSyncGet.Spec.Broker.Custom.URL
			Expect(url == URL).Should(BeTrue())

			By("Confirm the config matches the expected listener and pipeline configs")
			configMap := mesheSyncGet.Spec.WatchList
			Expect(configMap).ShouldNot(BeNil())
			expectedConfigMap := corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1apha1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "watch-list",
					Namespace: defaultNamespace,
				},
				Data: map[string]string{
					"blacklist":  "",
					whitelistKey: "[{\"Resource\":\"namespaces.v1.\",\"Events\":[\"ADDED\",\"DELETE\"]},{\"Resource\":\"replicasets.v1.apps\",\"Events\":[\"ADDED\",\"DELETE\"]},{\"Resource\":\"pods.v1.\",\"Events\":[\"MODIFIED\"]}]",
				},
			}
			Expect(configMap).To(Equal(expectedConfigMap))
		})

		It("The meshsync CRDs update the spec of the resources", func() {
			By(createFirstMsg)
			err := fakeClient.Create(ctx, meshSync)
			Expect(err).NotTo(HaveOccurred())

			By("Get the latest version of the resource")
			existing := &MeshSync{}
			err = fakeClient.Get(ctx, typeNamespace, existing)
			Expect(err).NotTo(HaveOccurred())

			By("Update the size of the meshsync CRDs")
			existing.Spec.Size = 5
			err = fakeClient.Update(ctx, existing, &client.UpdateOptions{FieldManager: "testcase-meshsync"})
			Expect(err).NotTo(HaveOccurred())

			By("Get the latest version of meshsync CRDs")
			meshSyncGet := &MeshSync{}
			err = fakeClient.Get(ctx, typeNamespace, meshSyncGet)
			Expect(err).NotTo(HaveOccurred())

			By("Confirm the size equal to 5")
			Expect(meshSyncGet.Spec.Size == 5).Should(BeTrue())
		})

		It("The meshsync CRDs update the status of the resources", func() {
			By(createFirstMsg)
			err := fakeClient.Create(ctx, meshSync)
			Expect(err).NotTo(HaveOccurred())

			By("Get the latest version of the resource")
			existing := &MeshSync{}
			err = fakeClient.Get(ctx, typeNamespace, existing)
			Expect(err).NotTo(HaveOccurred())

			existing.Status = MeshSyncStatus{
				PublishingTo: PublishingTo,
				Conditions: []metav1.Condition{
					{
						Type:               str,
						Status:             metav1.ConditionStatus(str),
						ObservedGeneration: 1,
						Reason:             Reason,
						Message:            Message,
						LastTransitionTime: metav1.Now(),
					},
				},
			}

			By("Update the status of the meshsync CRDs")
			err = fakeClient.Status().Update(ctx, existing)
			Expect(err).NotTo(HaveOccurred())
			Expect(existing.Status.PublishingTo == PublishingTo).Should(BeTrue())
		})

		It("The meshsyncList CRDs should be support by the kubernetes server", func() {
			By(createFirstMsg)
			err := fakeClient.Create(ctx, meshSync)
			Expect(err).NotTo(HaveOccurred())

			By("Confirm the meshsync CRDs support by the kubernetes server")
			meshSyncList := &MeshSyncList{}
			err = fakeClient.List(ctx, meshSyncList, &client.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(meshSyncList.Items) > 0).Should(BeTrue())
		})

	})

	Context("The test coverage for delete the meshsync CRDs", func() {
		It("The meshsync CRDs remove the resources", func() {
			By(createFirstMsg)
			err := fakeClient.Create(ctx, meshSync)
			Expect(err).NotTo(HaveOccurred())

			By("Delete the meshsync CRDs")
			err = fakeClient.Delete(ctx, meshSync)
			Expect(err).NotTo(HaveOccurred())

			By("Confirm deletion")
			deleted := &MeshSync{}
			err = fakeClient.Get(ctx, typeNamespace, deleted)
			Expect(apierrors.IsNotFound(err)).Should(BeTrue())
		})
	})
})

// customBrokerURLPatterns returns the generated pattern of
// spec.broker.custom.url for every served MeshSync version.
const (
	natsSchemePrefix         = "nats://"
	tlsSchemePrefix          = "tls://"
	crdSpecProp              = "spec"
	testBrokerDomainEndpoint = "broker.example.com:4222"
	testBrokerEndpoint       = "broker:4222"
	testLocalhostEndpoint    = "localhost:4222"
)

func customBrokerURLPatterns(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "config", "crd", "bases", "meshery.io_meshsyncs.yaml"))
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
			t.Fatalf("version %s: OpenAPI v3 schema missing", version.Name)
		}
		field := version.Schema.OpenAPIV3Schema.Properties[crdSpecProp].Properties["broker"].Properties["custom"].Properties["url"]
		if field.Pattern == "" {
			t.Fatalf("version %s: spec.broker.custom.url carries no pattern", version.Name)
		}
		patterns[version.Name] = field.Pattern
	}
	return patterns
}

// TestCustomBrokerURLPattern checks the NATS endpoint pattern the CRD enforces
// on spec.broker.custom.url. The pattern must be identical across all served
// versions, accept canonical nats://, tls://, and bare host:port endpoints with
// valid ports (1-65535), IPv4, bracketed IPv6, and optional userinfo, and reject
// invalid schemes (http, ftp), template placeholders, paths, and malformed inputs.
func TestCustomBrokerURLPattern(t *testing.T) {
	patterns := customBrokerURLPatterns(t)
	if len(patterns) < 2 {
		t.Fatalf("expected spec.broker.custom.url pattern in both v1alpha1 and v1alpha2, got %v", patterns)
	}
	var pattern string
	for version, p := range patterns {
		if pattern != "" && p != pattern {
			t.Fatalf("version %s pattern differs from the others; keep markers identical", version)
		}
		pattern = p
	}
	re := regexp.MustCompile(pattern)

	checkValidCustomBrokerURLs(t, re)
	checkInvalidCustomBrokerURLs(t, re)
	checkCustomBrokerURLProperties(t, re)
}

func parseCustomBrokerURL(raw string) (host string, port int, err error) {
	s := raw
	if strings.HasPrefix(s, natsSchemePrefix) {
		s = strings.TrimPrefix(s, natsSchemePrefix)
	} else if strings.HasPrefix(s, tlsSchemePrefix) {
		s = strings.TrimPrefix(s, tlsSchemePrefix)
	}
	if at := strings.LastIndex(s, "@"); at != -1 {
		s = s[at+1:]
	}
	var portStr string
	host, portStr, err = net.SplitHostPort(s)
	if err != nil {
		return "", 0, err
	}
	port, err = strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 || host == "" {
		return "", 0, fmt.Errorf("invalid host or port: %s:%s", host, portStr)
	}
	return host, port, nil
}

func checkValidCustomBrokerURLs(t *testing.T, re *regexp.Regexp) {
	t.Helper()
	valid := []string{
		natsSchemePrefix + testBrokerDomainEndpoint,
		tlsSchemePrefix + testBrokerDomainEndpoint,
		testBrokerDomainEndpoint,
		"broker.example.com.:4222",
		natsSchemePrefix + testBrokerEndpoint,
		tlsSchemePrefix + testBrokerEndpoint,
		testBrokerEndpoint,
		testLocalhostEndpoint,
		"localhost.:4222",
		natsSchemePrefix + testLocalhostEndpoint,
		tlsSchemePrefix + testLocalhostEndpoint,
		natsSchemePrefix + "meshery-nats:4222",
		natsSchemePrefix + "broker:04222",
		"broker:0080",
		natsSchemePrefix + "198.51.100.1:4222",
		tlsSchemePrefix + "192.168.1.100:80",
		"203.0.113.195:65535",
		natsSchemePrefix + "my-broker.meshery.svc.cluster.local:1",
		"my-broker.meshery.svc.cluster.local.:1",
		natsSchemePrefix + "[2001:db8::1]:4222",
		tlsSchemePrefix + "[::1]:4222",
		"[fe80::1]:4222",
		natsSchemePrefix + "[::ffff:192.0.2.1]:4222",
		natsSchemePrefix + "user:pass@broker:4222",
		tlsSchemePrefix + "token@broker.example.com:4222",
		"admin:secret@broker:4222",
		natsSchemePrefix + "client-app:p%40ssw0rd@192.168.1.5:4222",
		natsSchemePrefix + strings.Repeat("a", 63) + ".example.org:4222",
		strings.Repeat("a", 63) + ".example.org.:4222",
	}
	for _, value := range valid {
		host, port, err := parseCustomBrokerURL(value)
		if err != nil || host == "" || port < 1 || port > 65535 {
			t.Fatalf("fixture %q failed helper parsing: %v", value, err)
		}
		if !re.MatchString(value) {
			t.Errorf("pattern rejects valid custom broker URL %q", value)
		}
	}
}

func checkInvalidCustomBrokerURLs(t *testing.T, re *regexp.Regexp) {
	t.Helper()
	invalid := []string{
		"http://" + testBrokerDomainEndpoint,
		"https://meshery.io:4222",
		"https://meshery.io",
		"tcp://" + testBrokerDomainEndpoint,
		"ws://" + testBrokerDomainEndpoint,
		"wss://" + testBrokerDomainEndpoint,
		"ftp://" + testBrokerEndpoint,
		"nats://broker.example.com",
		"tls://broker.example.com",
		"broker.example.com",
		"198.51.100.1",
		":4222",
		natsSchemePrefix + ":4222",
		natsSchemePrefix + "broker:0",
		"broker:00",
		natsSchemePrefix + "broker:65536",
		"broker:065536",
		"broker:70000",
		natsSchemePrefix + testBrokerDomainEndpoint + "/",
		natsSchemePrefix + testBrokerDomainEndpoint + "/path",
		testBrokerDomainEndpoint + "/path",
		natsSchemePrefix + testBrokerEndpoint + "?ssl=true",
		"${PLAYGROUND_NATS_URL}",
		"${BROKER_URL}",
		"{{ .Values.broker.url }}",
		natsSchemePrefix + "broker..example.com:4222",
		natsSchemePrefix + "-broker:4222",
		"broker-:4222",
		natsSchemePrefix + "client-app:p@ssw0rd@192.168.1.5:4222",
		"2001:db8::1:4222",
		natsSchemePrefix + "2001:db8::1:4222",
		"[2001:db8::1]",
		natsSchemePrefix + "[2001:db8::1]",
		"[2001:db8:::1]:4222",
		"[::ffff:999.0.0.1]:4222",
		"[2001:db8::1]:0",
		"[2001:db8::1]:0000",
		natsSchemePrefix + strings.Repeat("a", 64) + ".example.org:4222",
		strings.Repeat("a", 64) + ".example.org.:4222",
		".:4222",
		" " + natsSchemePrefix + testBrokerEndpoint,
		natsSchemePrefix + testBrokerEndpoint + " ",
		" " + testBrokerEndpoint,
		"",
	}
	for _, value := range invalid {
		if re.MatchString(value) {
			t.Errorf("pattern accepts malformed custom broker URL %q", value)
		}
	}
}

func mutateCustomBrokerURL(rng *rand.Rand, value string) string {
	const alphabet = "0123456789abcdefABCDEF:./ g%@-_~"
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

func checkCustomBrokerURLProperties(t *testing.T, re *regexp.Regexp) {
	t.Helper()
	rng := rand.New(rand.NewSource(2)) //nolint:gosec // deterministic test data, not security
	for range 10000 {
		value := randomValidCustomBrokerURL(rng)
		if !re.MatchString(value) {
			t.Fatalf("pattern rejects valid custom broker URL %q", value)
		}
		mutated := mutateCustomBrokerURL(rng, value)
		if re.MatchString(mutated) {
			host, port, err := parseCustomBrokerURL(mutated)
			if err != nil {
				t.Fatalf("pattern accepts %q which parseCustomBrokerURL rejects: %v", mutated, err)
			}
			if port < 1 || port > 65535 || host == "" {
				t.Fatalf("pattern accepts %q with invalid port or empty host", mutated)
			}
			if strings.Contains(mutated, "[") && net.ParseIP(host) == nil {
				t.Fatalf("pattern accepts bracketed %q which net.ParseIP rejects", mutated)
			}
		}
	}
}

func randomValidCustomBrokerURL(rng *rand.Rand) string {
	port := 1 + rng.Intn(65535)
	var endpoint string
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
		endpoint = fmt.Sprintf("%s:%d", name, port)
	case 1:
		endpoint = fmt.Sprintf("%d.%d.%d.%d:%d", 1+rng.Intn(223), rng.Intn(256), rng.Intn(256), 1+rng.Intn(254), port)
	default:
		var raw [16]byte
		_, _ = rng.Read(raw[:])
		addr := netip.AddrFrom16(raw)
		endpoint = fmt.Sprintf("[%s]:%d", addr.String(), port)
	}

	var userinfo string
	switch rng.Intn(4) {
	case 1:
		userinfo = "token@"
	case 2:
		userinfo = "user:pass@"
	case 3:
		userinfo = "admin123:secret_token.456@"
	}

	var scheme string
	switch rng.Intn(3) {
	case 1:
		scheme = natsSchemePrefix
	case 2:
		scheme = tlsSchemePrefix
	}

	return scheme + userinfo + endpoint
}
