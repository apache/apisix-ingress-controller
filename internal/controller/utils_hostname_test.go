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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestListenersForGatewayContext(t *testing.T) {
	hostname := gatewayv1.Hostname("example.com")
	listener := gatewayv1.Listener{
		Name:     "http",
		Protocol: gatewayv1.HTTPProtocolType,
		Port:     gatewayv1.PortNumber(80),
		Hostname: &hostname,
	}

	tests := []struct {
		name     string
		context  RouteParentRefContext
		expected []gatewayv1.Listener
	}{
		{
			name: "prefer listeners slice when present",
			context: RouteParentRefContext{
				Listeners: []gatewayv1.Listener{
					listener,
					{
						Name:     "https",
						Protocol: gatewayv1.HTTPSProtocolType,
						Port:     gatewayv1.PortNumber(443),
					},
				},
				Listener: &gatewayv1.Listener{
					Name: "ignored",
				},
			},
			expected: []gatewayv1.Listener{
				listener,
				{
					Name:     "https",
					Protocol: gatewayv1.HTTPSProtocolType,
					Port:     gatewayv1.PortNumber(443),
				},
			},
		},
		{
			name: "fallback to single listener pointer",
			context: RouteParentRefContext{
				Listener: &listener,
			},
			expected: []gatewayv1.Listener{
				listener,
			},
		},
		{
			name:     "no matched listeners",
			context:  RouteParentRefContext{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, listenersForGatewayContext(tt.context))
		})
	}
}

func TestGetUnionOfGatewayHostnames(t *testing.T) {
	fooHostname := gatewayv1.Hostname("foo.example.com")
	barHostname := gatewayv1.Hostname("bar.example.com")

	t.Run("uses all matched listeners and ignores non-hostname-effective listeners", func(t *testing.T) {
		gateways := []RouteParentRefContext{
			{
				Listeners: []gatewayv1.Listener{
					{Name: "foo", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: &fooHostname},
					{Name: "bar", Protocol: gatewayv1.HTTPSProtocolType, Port: gatewayv1.PortNumber(443), Hostname: &barHostname},
					{Name: "tcp", Protocol: gatewayv1.TCPProtocolType, Port: gatewayv1.PortNumber(9100)},
				},
			},
		}

		hostnames, matchAny := getUnionOfGatewayHostnames(gateways)
		assert.False(t, matchAny)
		assert.Equal(t, []gatewayv1.Hostname{fooHostname, barHostname}, hostnames)
	})

	t.Run("listener without hostname matches any hostname", func(t *testing.T) {
		gateways := []RouteParentRefContext{
			{
				Listeners: []gatewayv1.Listener{
					{Name: "http", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80)},
				},
			},
		}

		hostnames, matchAny := getUnionOfGatewayHostnames(gateways)
		assert.True(t, matchAny)
		assert.Nil(t, hostnames)
	})
}

func TestGetMinimumHostnameIntersection(t *testing.T) {
	fooHostname := gatewayv1.Hostname("foo.example.com")
	routeHostname := gatewayv1.Hostname("foo.example.com")
	wildcardHostname := gatewayv1.Hostname("*.example.com")

	t.Run("matches across multiple listeners without sectionName", func(t *testing.T) {
		gateways := []RouteParentRefContext{
			{
				Listeners: []gatewayv1.Listener{
					{Name: "foo", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: &fooHostname},
					{Name: "wildcard", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: &wildcardHostname},
				},
			},
		}

		assert.Equal(t, routeHostname, getMinimumHostnameIntersection(gateways, routeHostname))
	})

	t.Run("returns empty when there is no listener match", func(t *testing.T) {
		gateways := []RouteParentRefContext{
			{
				Listeners: []gatewayv1.Listener{
					{Name: "foo", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: &fooHostname},
				},
			},
		}
		unmatched := gatewayv1.Hostname("bar.example.com")
		assert.Equal(t, gatewayv1.Hostname(""), getMinimumHostnameIntersection(gateways, unmatched))
	})
}

func TestFilterHostnamesWithMatchedListeners(t *testing.T) {
	fooHostname := gatewayv1.Hostname("foo.example.com")
	barHostname := gatewayv1.Hostname("bar.example.com")

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{
				fooHostname,
				barHostname,
			},
		},
	}

	gateways := []RouteParentRefContext{
		{
			Listeners: []gatewayv1.Listener{
				{Name: "foo", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: &fooHostname},
				{Name: "bar", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: &barHostname},
			},
		},
	}

	filtered, err := filterHostnames(gateways, route.DeepCopy())
	assert.NoError(t, err)
	assert.Equal(t, []gatewayv1.Hostname{fooHostname, barHostname}, filtered.Spec.Hostnames)
}

func TestFilterHostnamesNoMatchedListeners(t *testing.T) {
	fooHostname := gatewayv1.Hostname("foo.example.com")
	barHostname := gatewayv1.Hostname("bar.example.com")

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{
				barHostname,
			},
		},
	}

	gateways := []RouteParentRefContext{
		{
			Listeners: []gatewayv1.Listener{
				{Name: "foo", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: &fooHostname},
			},
		},
	}

	_, err := filterHostnames(gateways, route.DeepCopy())
	assert.ErrorIs(t, err, ErrNoMatchingListenerHostname)
}

func TestAppendUniqueListeners(t *testing.T) {
	listenerA := gatewayv1.Listener{Name: "a", Port: 80}
	listenerB := gatewayv1.Listener{Name: "b", Port: 81}
	listenerA2 := gatewayv1.Listener{Name: "a", Port: 82} // Duplicate name, different port

	tests := []struct {
		name     string
		target   []gatewayv1.Listener
		source   []gatewayv1.Listener
		expected []gatewayv1.Listener
	}{
		{
			name:     "empty target, add listeners",
			target:   nil,
			source:   []gatewayv1.Listener{listenerA, listenerB},
			expected: []gatewayv1.Listener{listenerA, listenerB},
		},
		{
			name:     "duplicate names skipped",
			target:   []gatewayv1.Listener{listenerA},
			source:   []gatewayv1.Listener{listenerA, listenerB},
			expected: []gatewayv1.Listener{listenerA, listenerB},
		},
		{
			name:     "mixed duplicates and new",
			target:   []gatewayv1.Listener{listenerA},
			source:   []gatewayv1.Listener{listenerB, listenerA, listenerA2},
			expected: []gatewayv1.Listener{listenerA, listenerB},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, appendUniqueListeners(tt.target, tt.source...))
		})
	}
}
