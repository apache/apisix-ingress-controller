// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeApisixRoutes implements ApisixRouteInterface
type FakeApisixRoutes struct {
	Fake *FakeApisixV2
	ns   string
}

var apisixroutesResource = v2.SchemeGroupVersion.WithResource("apisixroutes")

var apisixroutesKind = v2.SchemeGroupVersion.WithKind("ApisixRoute")

// Get takes name of the apisixRoute, and returns the corresponding apisixRoute object, and an error if there is any.
func (c *FakeApisixRoutes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v2.ApisixRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(apisixroutesResource, c.ns, name), &v2.ApisixRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2.ApisixRoute), err
}

// List takes label and field selectors, and returns the list of ApisixRoutes that match those selectors.
func (c *FakeApisixRoutes) List(ctx context.Context, opts v1.ListOptions) (result *v2.ApisixRouteList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(apisixroutesResource, apisixroutesKind, c.ns, opts), &v2.ApisixRouteList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v2.ApisixRouteList{ListMeta: obj.(*v2.ApisixRouteList).ListMeta}
	for _, item := range obj.(*v2.ApisixRouteList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested apisixRoutes.
func (c *FakeApisixRoutes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(apisixroutesResource, c.ns, opts))

}

// Create takes the representation of a apisixRoute and creates it.  Returns the server's representation of the apisixRoute, and an error, if there is any.
func (c *FakeApisixRoutes) Create(ctx context.Context, apisixRoute *v2.ApisixRoute, opts v1.CreateOptions) (result *v2.ApisixRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(apisixroutesResource, c.ns, apisixRoute), &v2.ApisixRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2.ApisixRoute), err
}

// Update takes the representation of a apisixRoute and updates it. Returns the server's representation of the apisixRoute, and an error, if there is any.
func (c *FakeApisixRoutes) Update(ctx context.Context, apisixRoute *v2.ApisixRoute, opts v1.UpdateOptions) (result *v2.ApisixRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(apisixroutesResource, c.ns, apisixRoute), &v2.ApisixRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2.ApisixRoute), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeApisixRoutes) UpdateStatus(ctx context.Context, apisixRoute *v2.ApisixRoute, opts v1.UpdateOptions) (*v2.ApisixRoute, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(apisixroutesResource, "status", c.ns, apisixRoute), &v2.ApisixRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2.ApisixRoute), err
}

// Delete takes name of the apisixRoute and deletes it. Returns an error if one occurs.
func (c *FakeApisixRoutes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(apisixroutesResource, c.ns, name, opts), &v2.ApisixRoute{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeApisixRoutes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(apisixroutesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v2.ApisixRouteList{})
	return err
}

// Patch applies the patch and returns the patched apisixRoute.
func (c *FakeApisixRoutes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v2.ApisixRoute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(apisixroutesResource, c.ns, name, pt, data, subresources...), &v2.ApisixRoute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v2.ApisixRoute), err
}
