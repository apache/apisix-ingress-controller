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

package conformance

import (
	"testing"

	"sigs.k8s.io/gateway-api/conformance"
	conformancev1 "sigs.k8s.io/gateway-api/conformance/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/tests"
)

// https://github.com/kubernetes-sigs/gateway-api/blob/5c5fc388829d24e8071071b01e8313ada8f15d9f/conformance/utils/suite/suite.go#L358.  SAN includes '*'
var skippedTestsForSSL = []string{
	tests.HTTPRouteHTTPSListener.ShortName,
	tests.HTTPRouteRedirectPortAndScheme.ShortName,
}

// APISIX terminates TLS on its stream proxy and matches stream routes by SNI,
// which implements TLSRoute in Terminate mode (declared via the
// TLSRouteModeTerminate feature) but never forwards the encrypted stream
// untouched. Every test below pins its listener to mode: Passthrough.
var skippedTestsForTLSPassthrough = []string{
	tests.TLSRouteSimpleSameNamespace.ShortName,
	tests.TLSRouteHostnameIntersection.ShortName,
	tests.TLSRouteInvalidBackendRefNonexistent.ShortName,
	tests.TLSRouteInvalidBackendRefUnknownKind.ShortName,
}

// Known gaps tracked for follow-up. These are genuine feature gaps rather than
// architectural limits, so they are expected to shrink over time.
var skippedTestsForKnownGaps = []string{
	// Listeners sharing a port but differing by hostname are not isolated from
	// each other yet, so requests fall through to a 404.
	tests.HTTPRouteListenerHostnameMatching.ShortName,
	tests.GRPCRouteListenerHostnameMatching.ShortName,

	// A backendRef that cannot be resolved must still produce a route that
	// answers 500; today no route is generated at all, so the request 404s.
	// A rule with omitted or empty backendRefs.
	tests.HTTPRouteNoBackendRefs.ShortName,
	// A backendRef of an unknown kind, which also needs ResolvedRefs=False.
	tests.HTTPRouteInvalidBackendRefUnknownKind.ShortName,
	// A cross-namespace backendRef with no ReferenceGrant. This one passes in
	// standalone mode and only fails against the APISIX admin API, so the two
	// provider paths disagree on how an unresolvable backend is synced.
	tests.HTTPRouteInvalidCrossNamespaceBackendRef.ShortName,
	// Terminate mode itself is covered by TLSRouteListenerTerminateSupportedKinds
	// and by the e2e TLSRoute suite. This provisional test additionally requires a
	// standalone Gateway with no GatewayProxy attached to reach Accepted=True, and
	// a stream proxy listening on the port it picks; neither holds here, so the
	// Gateway is rejected with "gateway proxy not found" before any traffic flows.
	tests.TLSRouteTerminateSimpleSameNamespace.ShortName,

	// A single HTTPRoute attached to several Gateways is not served from each
	// parent independently.
	tests.HTTPRouteMultipleGateways.ShortName,
}

func TestGatewayAPIConformance(t *testing.T) {
	opts := conformance.DefaultOptions(t)
	opts.Debug = true
	opts.CleanupBaseResources = true
	opts.GatewayClassName = gatewayClassName
	opts.SkipTests = append(opts.SkipTests, skippedTestsForSSL...)
	opts.SkipTests = append(opts.SkipTests, skippedTestsForTLSPassthrough...)
	opts.SkipTests = append(opts.SkipTests, skippedTestsForKnownGaps...)
	opts.Implementation = conformancev1.Implementation{
		Organization: "APISIX",
		Project:      "apisix-ingress-controller",
		URL:          "https://github.com/apache/apisix-ingress-controller.git",
		Version:      "v2.0.0",
		Contact:      []string{"https://github.com/apache/apisix-ingress-controller/issues"},
	}

	conformance.RunConformanceWithOptions(t, opts)
}
