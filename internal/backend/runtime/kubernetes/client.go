// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Unstructured client wrapper.
type client struct {
	client dynamic.Interface
	Mapper meta.RESTMapper
}

func (c *client) Resource(res *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	versions := []string{}

	gvk := res.GroupVersionKind()

	if gvk.Version != "" {
		versions = append(versions, gvk.Version)
	}

	mapping, err := c.Mapper.RESTMapping(gvk.GroupKind(), versions...)
	if err != nil {
		return nil, err
	}

	var dr dynamic.ResourceInterface

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = c.client.Resource(mapping.Resource).Namespace(res.GetNamespace())
	} else {
		dr = c.client.Resource(mapping.Resource)
	}

	return dr, nil
}

// Create saves the object obj in the Kubernetes cluster.
func (c *client) Create(ctx context.Context, res *unstructured.Unstructured, opts metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	dr, err := c.Resource(res)
	if err != nil {
		return nil, err
	}

	return dr.Create(ctx, res, opts, subresources...)
}

// Delete deletes the given obj from Kubernetes cluster.
func (c *client) Delete(ctx context.Context, resource, name, namespace string, opts metav1.DeleteOptions, subresources ...string) error {
	res, err := c.parseResource(resource, namespace)
	if err != nil {
		return err
	}

	dr, err := c.Resource(res)
	if err != nil {
		return err
	}

	return dr.Delete(ctx, name, opts, subresources...)
}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
func (c *client) Get(ctx context.Context, resource, name, namespace string, opts metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	res, err := c.parseResource(resource, namespace)
	if err != nil {
		return nil, err
	}

	dr, err := c.Resource(res)
	if err != nil {
		return nil, err
	}

	return dr.Get(ctx, name, opts, subresources...)
}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
func (c *client) List(ctx context.Context, resource, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	res, err := c.parseResource(resource, namespace)
	if err != nil {
		return nil, err
	}

	dr, err := c.Resource(res)
	if err != nil {
		return nil, err
	}

	return dr.List(ctx, opts)
}

// Update updates the resource.
func (c *client) Update(ctx context.Context, res *unstructured.Unstructured, opts metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	dr, err := c.Resource(res)
	if err != nil {
		return nil, err
	}

	return dr.Update(ctx, res, opts, subresources...)
}

func (c *client) kindFor(gvr schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return c.Mapper.KindFor(gvr)
}

func (c *client) parseResource(resource, namespace string) (*unstructured.Unstructured, error) {
	gvr, err := getGVR(resource)
	if err != nil {
		return nil, err
	}

	res := &unstructured.Unstructured{}

	gvk, err := c.kindFor(*gvr)
	if err != nil {
		return nil, err
	}

	res.SetGroupVersionKind(gvk)
	res.SetNamespace(namespace)

	return res, nil
}

func newClient(config *rest.Config) (*client, error) {
	mapper, err := apiutil.NewDynamicRESTMapper(config)
	if err != nil {
		return nil, err
	}

	c, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &client{c, mapper}, nil
}

func getGVR(resource string) (*schema.GroupVersionResource, error) {
	var gvr *schema.GroupVersionResource

	parts := strings.Split(resource, ".")

	if len(parts) == 2 {
		gvr = &schema.GroupVersionResource{
			Resource: parts[0],
			Version:  parts[1],
		}
	} else {
		gvr, _ = schema.ParseResourceArg(resource)
	}

	if gvr == nil {
		return nil, fmt.Errorf("couldn't parse resource name")
	}

	return gvr, nil
}
