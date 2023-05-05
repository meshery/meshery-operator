package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/layer5io/meshery-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// BrokersGetter has a method to return a BrokerInterface.
// A group's client should implement this interface.
type BrokersGetter interface {
	Brokers(namespace string) BrokerInterface
}

// BrokerInterface has methods to work with Broker resources.
type BrokerInterface interface {
	Create(ctx context.Context, broker *v1alpha1.Broker, opts metav1.CreateOptions) (*v1alpha1.Broker, error)
	Update(ctx context.Context, broker *v1alpha1.Broker, opts metav1.UpdateOptions) (*v1alpha1.Broker, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.Broker, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.BrokerList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.Broker, err error)
}

// broker implements BrokerInterface
type broker struct {
	client rest.Interface
	ns     string
}

// newBrokers returns a Brokers
func newBrokers(c *CoreClient, namespace string) *broker {
	return &broker{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the broker, and returns the corresponding broker object, and an error if there is any.
func (c *broker) Get(ctx context.Context, name string, opts metav1.GetOptions) (result *v1alpha1.Broker, err error) {
	result = &v1alpha1.Broker{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("brokers").
		Name(name).
		VersionedParams(&opts, ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Brokers that match those selectors.
func (c *broker) List(ctx context.Context, opts metav1.ListOptions) (result *v1alpha1.BrokerList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.BrokerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("brokers").
		VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested broker.
func (c *broker) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("brokers").
		VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a broker and creates it.  Returns the server's representation of the broker, and an error, if there is any.
func (c *broker) Create(ctx context.Context, broker *v1alpha1.Broker, opts metav1.CreateOptions) (result *v1alpha1.Broker, err error) {
	result = &v1alpha1.Broker{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("brokers").
		VersionedParams(&opts, ParameterCodec).
		Body(broker).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a broker and updates it. Returns the server's representation of the broker, and an error, if there is any.
func (c *broker) Update(ctx context.Context, broker *v1alpha1.Broker, opts metav1.UpdateOptions) (result *v1alpha1.Broker, err error) {
	result = &v1alpha1.Broker{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("brokers").
		Name(broker.Name).
		VersionedParams(&opts, ParameterCodec).
		Body(broker).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the broker and deletes it. Returns an error if one occurs.
func (c *broker) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("brokers").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched broker.
func (c *broker) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.Broker, err error) {
	result = &v1alpha1.Broker{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("brokers").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
