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
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	v2 "github.com/apache/apisix-ingress-controller/api/v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const DefaultIngressClassAnnotation = "ingressclass.kubernetes.io/is-default-class"

const (
	KindGateway              = "Gateway"
	KindHTTPRoute            = "HTTPRoute"
	KindGatewayClass         = "GatewayClass"
	KindIngress              = "Ingress"
	KindIngressClass         = "IngressClass"
	KindGatewayProxy         = "GatewayProxy"
	KindSecret               = "Secret"
	KindService              = "Service"
	KindApisixRoute          = "ApisixRoute"
	KindApisixGlobalRule     = "ApisixGlobalRule"
	KindApisixPluginConfig   = "ApisixPluginConfig"
	KindPod                  = "Pod"
	KindApisixTls            = "ApisixTls"
	KindApisixConsumer       = "ApisixConsumer"
	KindHTTPRoutePolicy      = "HTTPRoutePolicy"
	KindBackendTrafficPolicy = "BackendTrafficPolicy"
	KindConsumer             = "Consumer"
	KindPluginConfig         = "PluginConfig"
)

func KindOf(obj any) string {
	switch obj.(type) {
	case *gatewayv1.Gateway:
		return KindGateway
	case *gatewayv1.HTTPRoute:
		return KindHTTPRoute
	case *gatewayv1.GatewayClass:
		return KindGatewayClass
	case *v1.Ingress:
		return KindIngress
	case *v1.IngressClass:
		return KindIngressClass
	case *corev1.Secret:
		return KindSecret
	case *corev1.Service:
		return KindService
	case *v2.ApisixRoute:
		return KindApisixRoute
	case *v2.ApisixGlobalRule:
		return KindApisixGlobalRule
	case *v2.ApisixPluginConfig:
		return KindApisixPluginConfig
	case *corev1.Pod:
		return KindPod
	case *v2.ApisixTls:
		return KindApisixTls
	case *v2.ApisixConsumer:
		return KindApisixConsumer
	case *v1alpha1.HTTPRoutePolicy:
		return KindHTTPRoutePolicy
	case *v1alpha1.BackendTrafficPolicy:
		return KindBackendTrafficPolicy
	case *v1alpha1.GatewayProxy:
		return KindGatewayProxy
	case *v1alpha1.Consumer:
		return KindConsumer
	case *v1alpha1.PluginConfig:
		return KindPluginConfig
	default:
		return "Unknown"
	}
}
