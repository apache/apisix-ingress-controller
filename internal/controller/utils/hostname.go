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

package utils

import (
	"strings"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func HostnamesIntersect(a, b string) bool {
	return HostnamesMatch(a, b) || HostnamesMatch(b, a)
}

// HostnamesMatch checks that the hostnameB matches the hostnameA. HostnameA is treated as mask
// to be checked against the hostnameB.
func HostnamesMatch(hostnameA, hostnameB string) bool {
	// the hostnames are in the form of "foo.bar.com"; split them
	// in a slice of substrings
	hostnameALabels := strings.Split(hostnameA, ".")
	hostnameBLabels := strings.Split(hostnameB, ".")

	var a, b int
	var wildcard bool

	// iterate over the parts of both the hostnames
	for a, b = 0, 0; a < len(hostnameALabels) && b < len(hostnameBLabels); a, b = a+1, b+1 {
		var matchFound bool

		// if the current part of B is a wildcard, we need to find the first
		// A part that matches with the following B part
		if wildcard {
			for ; b < len(hostnameBLabels); b++ {
				if hostnameALabels[a] == hostnameBLabels[b] {
					matchFound = true
					break
				}
			}
		}

		// if no match was found, the hostnames don't match
		if wildcard && !matchFound {
			return false
		}

		// check if at least on of the current parts are a wildcard; if so, continue
		if hostnameALabels[a] == "*" {
			wildcard = true
			continue
		}
		// reset the wildcard  variables
		wildcard = false

		// if the current a part is different from the b part, the hostnames are incompatible
		if hostnameALabels[a] != hostnameBLabels[b] {
			return false
		}
	}
	return len(hostnameBLabels)-b == len(hostnameALabels)-a
}

// NarrowHostname returns every intersection of routeHostname with the hostnames
// of the given listeners.
//
// A listener carrying no hostname, or one whose hostname is a mask covering
// routeHostname, accepts routeHostname whole and is then the only intersection.
// Otherwise each listener hostname that routeHostname covers is an intersection
// of its own: keeping only the first - which is what looking up a single
// "smallest intersection" does - silently drops the traffic the remaining
// listeners accept.
//
// An empty result means the listeners accept none of routeHostname.
func NarrowHostname(listeners []gatewayv1.Listener, routeHostname gatewayv1.Hostname) []gatewayv1.Hostname {
	narrowed := make([]gatewayv1.Hostname, 0, len(listeners))
	for _, listener := range listeners {
		if listener.Hostname == nil || *listener.Hostname == "" ||
			HostnamesMatch(string(*listener.Hostname), string(routeHostname)) {
			return []gatewayv1.Hostname{routeHostname}
		}
		if HostnamesMatch(string(routeHostname), string(*listener.Hostname)) {
			narrowed = append(narrowed, *listener.Hostname)
		}
	}
	return narrowed
}

// UnionHostnames returns the union of the listener hostnames. The second return
// value reports that a listener carries no hostname of its own and therefore
// accepts every hostname, which makes the union meaningless.
func UnionHostnames(listeners []gatewayv1.Listener) ([]gatewayv1.Hostname, bool) {
	hostnames := make([]gatewayv1.Hostname, 0, len(listeners))
	for _, listener := range listeners {
		if listener.Hostname == nil || *listener.Hostname == "" {
			return nil, true
		}
		hostnames = append(hostnames, *listener.Hostname)
	}
	return hostnames, false
}

// DedupHostnames drops repeated hostnames, keeping the first occurrence.
// Listeners are aggregated across Gateways, so the same hostname can be
// contributed more than once.
func DedupHostnames(hostnames []gatewayv1.Hostname) []gatewayv1.Hostname {
	deduped := make([]gatewayv1.Hostname, 0, len(hostnames))
	seen := make(map[gatewayv1.Hostname]struct{}, len(hostnames))
	for _, hostname := range hostnames {
		if _, ok := seen[hostname]; ok {
			continue
		}
		seen[hostname] = struct{}{}
		deduped = append(deduped, hostname)
	}
	return deduped
}
