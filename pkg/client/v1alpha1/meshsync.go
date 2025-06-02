package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/meshery/meshery-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MeshSyncsGetter has a method to return a MeshSyncInterface.
// A group's client should implement this interface.
type MeshSyncsGetter interface {
	MeshSyncs(namespace string) MeshSyncInterface
}

// MeshSyncInterface has methods to work with MeshSync resources.
type MeshSyncInterface interface {
	Create(ctx context.Context, meshsync *v1alpha1.MeshSync, opts metav1.CreateOptions) (*v1alpha1.MeshSync, error)
	Update(ctx context.Context, meshsync *v1alpha1.MeshSync, opts metav1.UpdateOptions) (*v1alpha1.MeshSync, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1alpha1.MeshSync, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1alpha1.MeshSyncList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.MeshSync, err error)
}

// meshsync implements MeshSyncInterface
type meshsync struct {
	client rest.Interface
	ns     string
}

// newMeshSyncs returns a MeshSyncs
func newMeshSyncs(c *CoreClient, namespace string) *meshsync {
	return &meshsync{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the meshsync, and returns the corresponding meshsync object, and an error if there is any.
func (c *meshsync) Get(ctx context.Context, name string, opts metav1.GetOptions) (result *v1alpha1.MeshSync, err error) {
	result = &v1alpha1.MeshSync{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("meshsyncs").
		Name(name).
		VersionedParams(&opts, ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MeshSyncs that match those selectors.
func (c *meshsync) List(ctx context.Context, opts metav1.ListOptions) (result *v1alpha1.MeshSyncList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MeshSyncList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("meshsyncs").
		VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested meshsync.
func (c *meshsync) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("meshsyncs").
		VersionedParams(&opts, ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a meshsync and creates it.  Returns the server's representation of the meshsync, and an error, if there is any.
func (c *meshsync) Create(ctx context.Context, meshsync *v1alpha1.MeshSync, opts metav1.CreateOptions) (result *v1alpha1.MeshSync, err error) {
	result = &v1alpha1.MeshSync{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("meshsyncs").
		VersionedParams(&opts, ParameterCodec).
		Body(meshsync).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a meshsync and updates it. Returns the server's representation of the meshsync, and an error, if there is any.
func (c *meshsync) Update(ctx context.Context, meshsync *v1alpha1.MeshSync, opts metav1.UpdateOptions) (result *v1alpha1.MeshSync, err error) {
	result = &v1alpha1.MeshSync{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("meshsyncs").
		Name(meshsync.Name).
		VersionedParams(&opts, ParameterCodec).
		Body(meshsync).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the meshsync and deletes it. Returns an error if one occurs.
func (c *meshsync) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("meshsyncs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched meshsync.
func (c *meshsync) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1alpha1.MeshSync, err error) {
	result = &v1alpha1.MeshSync{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("meshsyncs").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
