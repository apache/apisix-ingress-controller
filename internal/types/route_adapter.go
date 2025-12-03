// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package types

import (
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type HTTPRouteAdapter struct {
	*gatewayv1.HTTPRoute
}

func (r HTTPRouteAdapter) GetParentStatuses() []gatewayv1.RouteParentStatus {
	return r.Status.Parents
}
func (r HTTPRouteAdapter) GetParentRefs() []gatewayv1.ParentReference {
	return r.Spec.ParentRefs
}
func (r HTTPRouteAdapter) GetObject() client.Object {
	return r.HTTPRoute
}

type GRPCRouteAdapter struct {
	*gatewayv1.GRPCRoute
}

func (r GRPCRouteAdapter) GetParentStatuses() []gatewayv1.RouteParentStatus {
	return r.Status.Parents
}
func (r GRPCRouteAdapter) GetParentRefs() []gatewayv1.ParentReference {
	return r.Spec.ParentRefs
}
func (r GRPCRouteAdapter) GetObject() client.Object {
	return r.GRPCRoute
}

type TCPRouteAdapter struct {
	*gatewayv1alpha2.TCPRoute
}

func (r TCPRouteAdapter) GetParentStatuses() []gatewayv1.RouteParentStatus {
	return r.Status.Parents
}

func (r TCPRouteAdapter) GetParentRefs() []gatewayv1.ParentReference {
	return r.Spec.ParentRefs
}
func (r TCPRouteAdapter) GetObject() client.Object {
	return r.TCPRoute
}

type UDPRouteAdapter struct {
	*gatewayv1alpha2.UDPRoute
}

func (r UDPRouteAdapter) GetParentStatuses() []gatewayv1.RouteParentStatus {
	return r.Status.Parents
}
func (r UDPRouteAdapter) GetParentRefs() []gatewayv1.ParentReference {
	return r.Spec.ParentRefs
}
func (r UDPRouteAdapter) GetObject() client.Object {
	return r.UDPRoute
}

type TLSRouteAdapter struct {
	*gatewayv1alpha2.TLSRoute
}

func (r TLSRouteAdapter) GetParentStatuses() []gatewayv1.RouteParentStatus {
	return r.Status.Parents
}
func (r TLSRouteAdapter) GetParentRefs() []gatewayv1.ParentReference {
	return r.Spec.ParentRefs
}
func (r TLSRouteAdapter) GetObject() client.Object {
	return r.TLSRoute
}

type RouteAdapter interface {
	client.Object
	GetParentStatuses() []gatewayv1.RouteParentStatus
	GetParentRefs() []gatewayv1.ParentReference
	GetObject() client.Object
}

func NewRouteAdapter(obj client.Object) RouteAdapter {
	switch r := obj.(type) {
	case *gatewayv1.HTTPRoute:
		return &HTTPRouteAdapter{HTTPRoute: r}
	case *gatewayv1.GRPCRoute:
		return &GRPCRouteAdapter{GRPCRoute: r}
	case *gatewayv1alpha2.TLSRoute:
		return &TLSRouteAdapter{TLSRoute: r}
	case *gatewayv1alpha2.TCPRoute:
		return &TCPRouteAdapter{TCPRoute: r}
	case *gatewayv1alpha2.UDPRoute:
		return &UDPRouteAdapter{UDPRoute: r}
	default:
		return nil
	}
}

func NewRouteListAdapter(objList client.ObjectList) []RouteAdapter {
	switch r := objList.(type) {
	case *gatewayv1.HTTPRouteList:
		return lo.Map(r.Items, func(item gatewayv1.HTTPRoute, _ int) RouteAdapter {
			return &HTTPRouteAdapter{HTTPRoute: &item}
		})
	case *gatewayv1.GRPCRouteList:
		return lo.Map(r.Items, func(item gatewayv1.GRPCRoute, _ int) RouteAdapter {
			return &GRPCRouteAdapter{GRPCRoute: &item}
		})
	case *gatewayv1alpha2.TLSRouteList:
		return lo.Map(r.Items, func(item gatewayv1alpha2.TLSRoute, _ int) RouteAdapter {
			return &TLSRouteAdapter{TLSRoute: &item}
		})
	case *gatewayv1alpha2.TCPRouteList:
		return lo.Map(r.Items, func(item gatewayv1alpha2.TCPRoute, _ int) RouteAdapter {
			return &TCPRouteAdapter{TCPRoute: &item}
		})
	case *gatewayv1alpha2.UDPRouteList:
		return lo.Map(r.Items, func(item gatewayv1alpha2.UDPRoute, _ int) RouteAdapter {
			return &UDPRouteAdapter{UDPRoute: &item}
		})
	default:
		return nil
	}
}
