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
package types

import (
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

const (
	KindTCPRoute  gatewayv1alpha2.Kind = "TCPRoute"
	KindTLSRoute  gatewayv1alpha2.Kind = "TLSRoute"
	KindHTTPRoute gatewayv1alpha2.Kind = "HTTPRoute"
	KindUDPRoute  gatewayv1alpha2.Kind = "UDPRoute"
)

type ListenerConf struct {
	// Gateway namespace
	Namespace string
	// Gateway name
	Name string

	SectionName string
	Protocol    gatewayv1alpha2.ProtocolType
	Port        gatewayv1alpha2.PortNumber
	Hostname    *gatewayv1alpha2.Hostname

	// namespace selector of AllowedRoutes
	RouteNamespace *gatewayv1alpha2.RouteNamespaces
	AllowedKinds   []gatewayv1alpha2.RouteGroupKind
}

func (c *ListenerConf) IsAllowedKind(r gatewayv1alpha2.RouteGroupKind) bool {
	if len(c.AllowedKinds) == 0 {
		return true
	}

	for _, c := range c.AllowedKinds {
		if *c.Group == *r.Group && c.Kind == r.Kind {
			return true
		}
	}
	return false
}

func (c *ListenerConf) IsHostnameMatch(hostnames []gatewayv1alpha2.Hostname) bool {
	if c.Hostname == nil || len(hostnames) == 0 {
		return true
	}
	for _, h := range hostnames {
		if !utils.IsHostnameMatch(string(*c.Hostname), string(h)) {
			return false
		}
	}
	return true
}

func (c *ListenerConf) HasHostname() bool {
	if c.Hostname == nil {
		return false
	}
	if string(*c.Hostname) == "" {
		return false
	}
	return true
}
