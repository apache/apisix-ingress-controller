// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func isGatewayManaged(ctx context.Context, c client.Client, gateway *gatewayv1.Gateway) (bool, error) {
	if gateway == nil {
		return false, nil
	}

	className := string(gateway.Spec.GatewayClassName)
	if className == "" {
		return false, nil
	}

	var gatewayClass gatewayv1.GatewayClass
	if err := c.Get(ctx, client.ObjectKey{Name: className}, &gatewayClass); err != nil {
		if client.IgnoreNotFound(err) == nil {
			return false, nil
		}
		return false, err
	}

	return string(gatewayClass.Spec.ControllerName) == config.ControllerConfig.ControllerName, nil
}

func isHTTPRouteManaged(ctx context.Context, c client.Client, route *gatewayv1.HTTPRoute) (bool, error) {
	if route == nil {
		return false, nil
	}
	return routeReferencesManagedGateway(ctx, c, route.Spec.ParentRefs, route.Namespace)
}

func isGRPCRouteManaged(ctx context.Context, c client.Client, route *gatewayv1.GRPCRoute) (bool, error) {
	if route == nil {
		return false, nil
	}
	return routeReferencesManagedGateway(ctx, c, route.Spec.ParentRefs, route.Namespace)
}

func isTCPRouteManaged(ctx context.Context, c client.Client, route *gatewayv1alpha2.TCPRoute) (bool, error) {
	if route == nil {
		return false, nil
	}
	return routeReferencesManagedGateway(ctx, c, route.Spec.ParentRefs, route.Namespace)
}

func routeReferencesManagedGateway(ctx context.Context, c client.Client, parents []gatewayv1.ParentReference, defaultNamespace string) (bool, error) {
	for _, parent := range parents {
		if parent.Name == "" {
			continue
		}
		if parent.Kind != nil && string(*parent.Kind) != internaltypes.KindGateway {
			continue
		}

		namespace := defaultNamespace
		if parent.Namespace != nil && *parent.Namespace != "" {
			namespace = string(*parent.Namespace)
		}

		var gateway gatewayv1.Gateway
		if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: string(parent.Name)}, &gateway); err != nil {
			if client.IgnoreNotFound(err) == nil {
				continue
			}
			return false, err
		}

		managed, err := isGatewayManaged(ctx, c, &gateway)
		if err != nil {
			return false, err
		}
		if managed {
			return true, nil
		}
	}

	return false, nil
}
