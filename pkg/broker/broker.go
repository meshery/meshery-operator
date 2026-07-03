package broker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	mesheryv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Object interface {
	runtime.Object
	metav1.Object
}

// GetObjects returns the NATS server objects for a Broker: the official chart's
// vendored, embedded manifests (ConfigMap, headless + client Services,
// StatefulSet) deep-copied into the Broker's namespace, with a small overlay
// mapping BrokerSpec onto them. The order is deterministic (chart render order:
// config and Services precede the StatefulSet), which suits Server-Side Apply.
//
// The auth Secret is NOT included here — it carries a generated token that must
// be stable across reconciles, so it is managed separately (BuildAuthSecret +
// the controller's ensure-once logic).
func GetObjects(m *mesheryv1alpha1.Broker) []Object {
	objs := make([]Object, 0, len(natsTemplate))
	for _, tmpl := range natsTemplate {
		obj := tmpl.DeepCopyObject().(Object)
		obj.SetNamespace(m.Namespace)
		overlayBrokerSpec(obj, m)
		objs = append(objs, obj)
	}
	return objs
}

// desiredReplicas defaults an unset spec.size to one replica so an omitted
// size never applies a zero-replica StatefulSet that CheckHealth (which
// expects one ready replica) would report unhealthy forever.
func desiredReplicas(m *mesheryv1alpha1.Broker) int32 {
	if m.Spec.Size > 0 {
		return m.Spec.Size
	}
	return 1
}

// overlayBrokerSpec maps the reconcilable BrokerSpec fields onto the vendored
// objects: Size -> StatefulSet replicas, Version -> NATS image tag, and
// Service.* -> the client Service (the headless Service is left untouched).
func overlayBrokerSpec(obj Object, m *mesheryv1alpha1.Broker) {
	switch o := obj.(type) {
	case *v1.StatefulSet:
		if o.Name != natsServiceName {
			return
		}
		replicas := desiredReplicas(m)
		o.Spec.Replicas = &replicas
		if m.Spec.Version != "" {
			for i := range o.Spec.Template.Spec.Containers {
				if o.Spec.Template.Spec.Containers[i].Name == natsName {
					o.Spec.Template.Spec.Containers[i].Image = "nats:" + m.Spec.Version
				}
			}
		}
	case *corev1.Service:
		// Only the client Service (natsServiceName); the headless Service is
		// "<name>-headless" and must stay ClusterIP: None.
		if o.Name == natsServiceName {
			applyServiceSpec(o, m.Spec.Service)
		}
	}
}

// applyServiceSpec overlays the user-declared networking onto the NATS client
// Service. An unset Type keeps the chart's rendered default (ClusterIP) — the
// old implicit LoadBalancer default is gone (#801): a broker should not acquire
// a public address unless the CR asks for one. LoadBalancer-only fields are
// applied only for that type.
func applyServiceSpec(obj *corev1.Service, svc mesheryv1alpha1.BrokerServiceSpec) {
	if svc.Type != "" {
		obj.Spec.Type = svc.Type
	}
	for k, v := range svc.Annotations {
		if obj.Annotations == nil {
			obj.Annotations = map[string]string{}
		}
		obj.Annotations[k] = v
	}
	if obj.Spec.Type == corev1.ServiceTypeLoadBalancer {
		obj.Spec.LoadBalancerClass = svc.LoadBalancerClass
		obj.Spec.LoadBalancerSourceRanges = svc.LoadBalancerSourceRanges
	}
}

// tokenPrefix guarantees the generated token starts with a letter. The token is
// injected into nats.conf unquoted (`token: $NATS_TOKEN`, see
// pkg/broker/chart/values.yaml), and NATS re-lexes the substituted value. A bare
// hex token that starts with digits and contains an `e` (e.g. "758e126b...") is
// misparsed as scientific notation ("7.58e126") and the NATS server crashes on
// startup with a config parse error, leaving the Broker permanently NotReady.
// Quoting the value in the config is not an option because that disables NATS's
// env-var expansion, so we make the token safe by construction: a leading letter
// forces NATS to lex the whole token as a string.
const tokenPrefix = "t"

// GenerateToken returns a cryptographically random token for NATS auth, safe to
// embed unquoted in nats.conf (always lexes as a string — see tokenPrefix).
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return tokenPrefix + hex.EncodeToString(b), nil
}

// BuildAuthSecret constructs the meshery-nats-auth Secret holding the NATS token
// the server reads via $NATS_TOKEN and clients present to connect.
func BuildAuthSecret(namespace, token string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      AuthSecretName,
			Namespace: namespace,
			Labels:    BrokerLabel,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{"token": token},
	}
}

// CheckHealth reports whether the NATS StatefulSet has reached its desired ready
// replica count. StatefulSets do not populate status.conditions reliably, so
// ReadyReplicas is the authoritative signal.
func CheckHealth(ctx context.Context, m *mesheryv1alpha1.Broker, c client.Client) error {
	obj := &v1.StatefulSet{}
	if err := c.Get(ctx, types.NamespacedName{Name: natsServiceName, Namespace: m.Namespace}, obj); err != nil {
		return ErrGettingBrokerResource(err)
	}

	desired := desiredReplicas(m)
	if obj.Status.ReadyReplicas != desired {
		return ErrBrokerReplicasNotReady(fmt.Sprintf("%d of %d replicas ready", obj.Status.ReadyReplicas, desired))
	}
	return nil
}

// GetEndpoint derives the broker endpoints from the client Service and writes
// them to the Broker status. Derivation is pure and non-blocking: it reads only
// the Service spec/status (no TCP dials). Addresses are stored as host:port; the
// nats:// scheme (and any token) is applied where the endpoint is consumed.
func GetEndpoint(ctx context.Context, m *mesheryv1alpha1.Broker, c client.Client, apiServerURL string) error {
	serviceObj := &corev1.Service{}
	if err := c.Get(ctx, types.NamespacedName{Name: natsServiceName, Namespace: m.Namespace}, serviceObj); err != nil {
		return ErrGettingBrokerResource(err)
	}

	internal, external, _ := DeriveEndpoint(serviceObj, apiServerURL)
	// An explicit override wins over auto-derivation (ingress/gateway, air-gapped,
	// NAT topologies where TCP-derived external addresses are wrong).
	if override := m.Spec.Service.ExternalEndpointOverride; override != "" {
		external = override
	}
	m.Status.Endpoint.Internal = internal
	m.Status.Endpoint.External = external
	return nil
}
