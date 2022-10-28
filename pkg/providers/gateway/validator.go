// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package gateway

import (
	"context"
	"fmt"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/gateway/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type Validator struct {
	provider *Provider
}

func newValidator(p *Provider) *Validator {
	return &Validator{
		provider: p,
	}
}

type commonRoute struct {
	routeNamespace string
	parentRefs     []gatewayv1alpha2.ParentRef
	routeProtocol  gatewayv1alpha2.ProtocolType
	routeHostnames []gatewayv1alpha2.Hostname
	routeGroupKind gatewayv1alpha2.RouteGroupKind
}

func (r *commonRoute) hasParentRefs() bool {
	return len(r.parentRefs) != 0
}

func (r *commonRoute) isHTTPProtocol() bool {
	return r.routeProtocol == gatewayv1alpha2.HTTPProtocolType
}

func (r *commonRoute) isHTTPSProtocol() bool {
	return r.routeProtocol == gatewayv1alpha2.HTTPSProtocolType
}

func parseToCommentRoute(route any) (*commonRoute, error) {
	r := new(commonRoute)
	group := gatewayv1alpha2.Group(gatewayv1alpha2.GroupName)
	switch route := route.(type) {
	case *gatewayv1alpha2.HTTPRoute:
		r.routeNamespace = route.Namespace
		r.parentRefs = route.Spec.ParentRefs
		r.routeProtocol = gatewayv1alpha2.HTTPProtocolType
		r.routeHostnames = route.Spec.Hostnames
		r.routeGroupKind = gatewayv1alpha2.RouteGroupKind{
			Group: &group,
			Kind:  types.KindHTTPRoute,
		}
	case *gatewayv1alpha2.TLSRoute:
		r.routeNamespace = route.Namespace
		r.parentRefs = route.Spec.ParentRefs
		r.routeProtocol = gatewayv1alpha2.HTTPSProtocolType
		r.routeHostnames = route.Spec.Hostnames
		r.routeGroupKind = gatewayv1alpha2.RouteGroupKind{
			Group: &group,
			Kind:  types.KindTLSRoute,
		}
	case *gatewayv1alpha2.TCPRoute:
		r.routeNamespace = route.Namespace
		r.parentRefs = route.Spec.ParentRefs
		r.routeProtocol = gatewayv1alpha2.TCPProtocolType
		r.routeGroupKind = gatewayv1alpha2.RouteGroupKind{
			Group: &group,
			Kind:  types.KindTCPRoute,
		}
	case *gatewayv1alpha2.UDPRoute:
		r.routeNamespace = route.Namespace
		r.parentRefs = route.Spec.ParentRefs
		r.routeProtocol = gatewayv1alpha2.UDPProtocolType
		r.routeGroupKind = gatewayv1alpha2.RouteGroupKind{
			Group: &group,
			Kind:  types.KindUDPRoute,
		}
	default:
		return nil, fmt.Errorf("validator unsupported Route")
	}
	return r, nil
}

func (v *Validator) getListenersConf(r *commonRoute, parentRef gatewayv1alpha2.ParentRef) ([]*types.ListenerConf, error) {
	var name, kind, namespace, sectionName string
	name = string(parentRef.Name)
	if parentRef.Kind != nil {
		kind = string(*parentRef.Kind)
	} else {
		kind = "Gateway"
	}
	if parentRef.Namespace != nil {
		namespace = string(*parentRef.Namespace)
	}
	if parentRef.SectionName != nil {
		sectionName = string(*parentRef.SectionName)
	}

	// The only kind of parent resource with "Core" support is Gateway.
	if kind != "Gateway" {
		return nil, fmt.Errorf("ParentRef.Kind support Gateway only")
	}
	//  When namespace unspecified this refers to the local namespace of the Route.
	if namespace == "" {
		namespace = r.routeNamespace
	}

	listeners := make([]*types.ListenerConf, 0)
	if sectionName != "" {
		listenerConf, err := v.provider.FindListener(namespace, name, sectionName)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listenerConf)
	} else {
		_listeners, err := v.provider.QueryListeners(namespace, name)
		if err != nil {
			return nil, err
		}
		for _, listenerConf := range _listeners {
			listeners = append(listeners, listenerConf)
		}
	}
	if len(listeners) == 0 {
		log.Warnw("can't find ListenerConf by ParentRef",
			zap.Any("ParentRef", parentRef),
		)
		return nil, fmt.Errorf("can't find Listener by ParentRef")
	}
	return listeners, nil
}

func (v *Validator) validateParentRefV1Alpha2(r *commonRoute) ([]*types.ListenerConf, error) {
	var matchedListeners []*types.ListenerConf
	for _, parentRef := range r.parentRefs {
		listeners, err := v.getListenersConf(r, parentRef)
		if err != nil {
			return nil, err
		}

		// filter listener by ParentRef
		for _, listenerConf := range listeners {

			if listenerConf.Protocol != r.routeProtocol {
				continue
			}
			if !listenerConf.IsAllowedKind(r.routeGroupKind) {
				// TODO: set the “ResolvedRefs” condition to False for this Listener with the “InvalidRouteKinds” reason.
				continue
			}
			if r.isHTTPProtocol() || r.isHTTPSProtocol() {
				if listenerConf.HasHostname() && len(r.routeHostnames) != 0 {
					if !listenerConf.IsHostnameMatch(r.routeHostnames) {
						continue
					}
				}
			}

			// match listener by AllowRoute.Namespaces
			switch *listenerConf.RouteNamespace.From {
			case gatewayv1alpha2.NamespacesFromSame:
				if r.routeNamespace != listenerConf.Namespace {
					continue
				}
			case gatewayv1alpha2.NamespacesFromSelector:
				// get listener namespace with selector labeled namespace
				selector, err := metav1.LabelSelectorAsSelector(listenerConf.RouteNamespace.Selector)
				if err != nil {
					log.Errorw("convert Selector failed",
						zap.Error(err),
						zap.Any("Object", listenerConf.RouteNamespace.Selector),
					)
					return nil, err
				}
				allowedNamespaces := &corev1.NamespaceList{}
				err = v.provider.runtimeClient.List(
					context.TODO(), allowedNamespaces,
					&runtimeclient.ListOptions{LabelSelector: selector},
				)
				if err != nil {
					log.Errorw("list parent namespace failed",
						zap.Error(err),
						zap.Any("Object", *listenerConf.RouteNamespace.From),
					)
					return nil, err
				}
				namespaceMatched := false
				for _, allowedNamespace := range allowedNamespaces.Items {
					if string(allowedNamespace.Name) == r.routeNamespace {
						namespaceMatched = true
						break
					}
				}
				if !namespaceMatched {
					continue
				}
			}

			matchedListeners = append(matchedListeners, listenerConf)
		}
	}
	if len(matchedListeners) == 0 {
		log.Errorw("no listeners referenced by ParentRefs",
			zap.Any("listeners", v.provider.listeners),
			zap.Any("route", r),
		)
		return nil, fmt.Errorf("no listeners referenced by ParentRefs")
	}
	return matchedListeners, nil
}

// ValidateCommonRouteV1Alpha2 only checks CommonRoute and ParentRef logic.
// route support HTTPRoute TLSRoute UDPRoute TLSRoute for now.
func (v *Validator) ValidateCommonRouteV1Alpha2(route any) error {
	r, err := parseToCommentRoute(route)
	if err != nil {
		return err
	}

	if r.hasParentRefs() {
		_, err = v.validateParentRefV1Alpha2(r)
		if err != nil {
			return err
		}
	}
	return nil
}
