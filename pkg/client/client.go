package client

import (
	apiv1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	v1alpha1 "github.com/meshery/meshery-operator/pkg/client/v1alpha1"

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
	SchemeGroupVersion = apiv1alpha1.GroupVersion
)

type Interface interface {
	CoreV1Alpha1() v1alpha1.CoreInterface
}

type Clientset struct {
	corev1alpha1 *v1alpha1.CoreClient
}

// CoreV1Alpha1 retrieves the CoreV1Alpha1Client
func (c *Clientset) CoreV1Alpha1() v1alpha1.CoreInterface {
	return c.corev1alpha1
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
		corev1alpha1: v1alpha1.New(client, runtime.NewParameterCodec(Scheme)),
	}, nil
}

func init() {
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	// +kubebuilder:scaffold:scheme
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(apiv1alpha1.AddToScheme(Scheme))
}
