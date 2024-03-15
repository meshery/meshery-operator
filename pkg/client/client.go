package client

import (
	apiv1beta1 "github.com/layer5io/meshery-operator/api/v1beta1"
	v1beta1 "github.com/layer5io/meshery-operator/pkg/client/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var (
	Scheme             = runtime.NewScheme()
	SchemeGroupVersion = apiv1beta1.GroupVersion
)

type Interface interface {
	Corev1Beta1() v1beta1.CoreInterface
}

type Clientset struct {
	corev1beta1 *v1beta1.CoreClient
}

// Corev1beta1 retrieves the Corev1beta1Client
func (c *Clientset) Corev1Beta1() v1beta1.CoreInterface {
	return c.corev1beta1
}

func New(config *rest.Config) (Interface, error) {
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.NewCodecFactory(Scheme).WithoutConversion()

	client, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &Clientset{
		corev1beta1: v1beta1.New(client, runtime.NewParameterCodec(Scheme)),
	}, nil
}

func init() {
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	// +kubebuilder:scaffold:scheme
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(apiv1beta1.AddToScheme(Scheme))
}
