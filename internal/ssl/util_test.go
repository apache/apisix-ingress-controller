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

package ssl

import "testing"

func TestHostsOverlap(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"example.com", "example.com", true},        // identical exact
		{"*.example.com", "*.example.com", true},    // identical wildcard
		{"app.example.com", "*.example.com", true},  // exact covered by wildcard
		{"*.example.com", "app.example.com", true},  // wildcard covers exact (inverse)
		{"App.Example.com", "*.EXAMPLE.com", true},  // case-insensitive
		{"a.b.example.com", "*.example.com", false}, // multi-label not covered
		{"example.com", "*.example.com", false},     // apex not covered by its wildcard
		{"app.example.com", "app.other.com", false}, // different exacts
		{"*.example.com", "*.other.com", false},     // distinct wildcards never overlap
		{"*.a.example.com", "*.example.com", false}, // wildcards at different depths
		{"", "example.com", false},                  // empty
	}
	for _, c := range cases {
		if got := HostsOverlap(c.a, c.b); got != c.want {
			t.Errorf("HostsOverlap(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestHostCoveredBy(t *testing.T) {
	cases := []struct {
		host, san string
		want      bool
	}{
		{"app.example.com", "app.example.com", true},       // exact SAN covers exact host
		{"app.example.com", "*.example.com", true},         // wildcard SAN covers subdomain
		{"App.Example.com", "*.EXAMPLE.com", true},         // case-insensitive
		{"*.example.com", "*.example.com", true},           // wildcard host needs identical wildcard SAN
		{"*.example.com", "app.example.com", false},        // exact SAN can't cover a wildcard host
		{"a.b.example.com", "*.example.com", false},        // wildcard SAN is single-label only
		{"shop.example.com", "internal.corp.local", false}, // unrelated SAN
		{"example.com", "*.example.com", false},            // apex not covered by its wildcard
		{"app.example.com", "", false},                     // empty SAN
	}
	for _, c := range cases {
		if got := HostCoveredBy(c.host, c.san); got != c.want {
			t.Errorf("HostCoveredBy(%q, %q) = %v, want %v", c.host, c.san, got, c.want)
		}
	}
}

func TestParentWildcard(t *testing.T) {
	cases := []struct {
		host, want string
	}{
		{"app.example.com", "*.example.com"},
		{"example.com", "*.com"},
		{"*.example.com", ""}, // already a wildcard
		{"localhost", ""},     // no parent label
		{"", ""},
	}
	for _, c := range cases {
		if got := ParentWildcard(c.host); got != c.want {
			t.Errorf("ParentWildcard(%q) = %q, want %q", c.host, got, c.want)
		}
	}
}
