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

package translator

import (
	"github.com/go-logr/logr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

type Translator struct {
	Log                   logr.Logger
	ListenerPortMatchMode config.ListenerPortMatchMode
}

// normalizeMode resolves an unset or unrecognised mode to the default. Emitting a
// server_port predicate is only correct once an operator has confirmed the data
// plane listens on the Gateway's declared ports, so anything unclear falls back
// to off rather than to a mode that injects.
func normalizeMode(mode config.ListenerPortMatchMode) config.ListenerPortMatchMode {
	switch mode {
	case config.ListenerPortMatchModeAuto, config.ListenerPortMatchModeExplicit, config.ListenerPortMatchModeOff:
		return mode
	default:
		return config.ListenerPortMatchModeOff
	}
}

func NewTranslator(log logr.Logger, mode config.ListenerPortMatchMode) *Translator {
	return &Translator{
		Log:                   log.WithName("translator"),
		ListenerPortMatchMode: normalizeMode(mode),
	}
}

// collectServerPortMatchPorts returns the hostname-less listener ports, which is
// the set used to decide whether a server_port var is needed at all.
//
// Listeners carrying a hostname are isolated by that hostname (service.hosts),
// which is the correct discriminator when several listeners share a single port.
// A server_port var adds no isolation for them and actively breaks routing: it
// pins the route to the Gateway's declared listener port, which need not equal
// the port APISIX actually accepts the connection on (node_listen), turning
// every request into a 404. Only hostname-less listeners rely on port-based
// isolation, so only their ports drive the decision to inject.
func collectServerPortMatchPorts(listeners []gatewayv1.Listener) map[int32]struct{} {
	ports := make(map[int32]struct{})
	for _, listener := range listeners {
		if listener.Hostname != nil && *listener.Hostname != "" {
			continue
		}
		ports[listener.Port] = struct{}{}
	}
	return ports
}

// allListenerPorts returns every targeted listener port. Once a server_port var
// is emitted it is applied to all of the route's APISIX routes, so it must list
// every port the route is attached to - including hostname listeners. Otherwise
// a route bound to both a hostname-less and a hostname listener would carry a
// predicate for the hostname-less port only, silently dropping traffic that
// arrives through the hostname listener's port.
func allListenerPorts(listeners []gatewayv1.Listener) map[int32]struct{} {
	ports := make(map[int32]struct{})
	for _, listener := range listeners {
		ports[listener.Port] = struct{}{}
	}
	return ports
}

// listenerScheme returns the request scheme shared by every listener the route
// attached to, or "" when they disagree, when there are none, or when any of
// them is a protocol that carries no request scheme.
//
// Only an unambiguous answer pins the route. A route attached to both an HTTP and
// an HTTPS listener is meant to serve both, and a listener protocol that has no
// scheme at all - TLS, TCP, UDP, which carry the L4 route kinds - leaves the
// route alone rather than being guessed at.
func listenerScheme(listeners []gatewayv1.Listener) string {
	scheme := ""
	for _, listener := range listeners {
		var current string
		switch listener.Protocol {
		case gatewayv1.HTTPProtocolType:
			current = apiv2.SchemeHTTP
		case gatewayv1.HTTPSProtocolType:
			current = apiv2.SchemeHTTPS
		default:
			return ""
		}
		if scheme != "" && scheme != current {
			return ""
		}
		scheme = current
	}
	return scheme
}

// pinRoutesToListenerScheme pins the routes of one rule to the scheme their
// listeners accept, when the listeners agree on one.
func (t *Translator) pinRoutesToListenerScheme(listeners []gatewayv1.Listener, routes []*adctypes.Route) {
	scheme := listenerScheme(listeners)
	if scheme == "" {
		return
	}
	for _, route := range routes {
		addSchemeVar(route, scheme)
	}
}

// addSchemeVar pins a route to the scheme of the connection APISIX accepted.
//
// Unlike server_port this holds whatever port mapping sits in front of the data
// plane, because $scheme reflects the connection itself rather than a number the
// Gateway declared. It is therefore independent of listener_port_match_mode,
// which exists to pin a route to a listener port and cannot isolate protocols
// unless the declared ports happen to match the ones APISIX listens on.
func addSchemeVar(route *adctypes.Route, scheme string) {
	route.Vars = append(route.Vars, []adctypes.StringOrSlice{
		{StrVal: "scheme"},
		{StrVal: "=="},
		{StrVal: scheme},
	})
}

// shouldInjectServerPortVars decides whether to pin the route to the matched
// listener port(s) via a server_port predicate.
//
// explicit reports whether the route attached to its Gateway through an explicit
// sectionName or port. It is computed by the controller from the matched
// RouteParentRefContext (provider.TranslateContext.HasExplicitListenerMatch),
// where each parentRef's Gateway and matched listeners are known, so an invalid
// explicit ref on one Gateway can never be satisfied by a same-named/ported
// listener matched through a different parentRef's Gateway.
func (t *Translator) shouldInjectServerPortVars(explicit bool, ports map[int32]struct{}) bool {
	if len(ports) == 0 {
		return false
	}

	switch t.ListenerPortMatchMode {
	case config.ListenerPortMatchModeOff:
		if explicit {
			t.Log.V(1).Info("listener_port_match_mode is 'off'; ignoring explicit listener targeting")
		}
		return false
	case config.ListenerPortMatchModeExplicit:
		return explicit
	case config.ListenerPortMatchModeAuto:
		return explicit || len(ports) > 1
	default:
		return false
	}
}

type TranslateResult struct {
	Services       []*adctypes.Service
	SSL            []*adctypes.SSL
	GlobalRules    adctypes.GlobalRule
	PluginMetadata adctypes.PluginMetadata
	Consumers      []*adctypes.Consumer
}
