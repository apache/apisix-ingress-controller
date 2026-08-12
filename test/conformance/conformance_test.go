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
	// An unresolvable or unknown-kind backendRef must respond 500. The
	// translator already injects fault-injection for these, but they still trip
	// against the APISIX admin API: UnknownKind fails consistently (passes in
	// standalone), while the cross-namespace and nonexistent cases are flaky
	// across runs. Kept skipped until the empty-upstream sync is sorted out.
	tests.HTTPRouteInvalidBackendRefUnknownKind.ShortName,
	tests.HTTPRouteInvalidCrossNamespaceBackendRef.ShortName,
	tests.HTTPRouteInvalidNonExistentBackendRef.ShortName,
	// Terminate mode itself is covered by TLSRouteListenerTerminateSupportedKinds
	// and by the e2e TLSRoute suite. This provisional test additionally requires a
	// standalone Gateway with no GatewayProxy attached to reach Accepted=True, and
	// a stream proxy listening on the port it picks; neither holds here, so the
	// Gateway is rejected with "gateway proxy not found" before any traffic flows.
	tests.TLSRouteTerminateSimpleSameNamespace.ShortName,

	// A single HTTPRoute attached to several Gateways is not served from each
	// parent independently.
	tests.HTTPRouteMultipleGateways.ShortName,

	// Gateway API 1.6 added these two, and the status semantics they ask for
	// were never implemented: nothing in the controller sets InvalidParameters
	// for a parametersRef it cannot resolve, or UnsupportedProtocol on a
	// listener whose protocol it does not know, or ListenersNotValid on the
	// Gateway. They are skipped rather than left failing so the report says
	// skipped instead of claiming a pass.
	tests.GatewayInvalidParametersRef.ShortName,
	tests.GatewayListenerUnsupportedProtocol.ShortName,
}

func TestGatewayAPIConformance(t *testing.T) {
	opts := conformance.DefaultOptions(t)
	opts.Debug = true
	opts.CleanupBaseResources = true
	opts.GatewayClassName = gatewayClassName
	opts.SkipTests = append(opts.SkipTests, skippedTestsForSSL...)
	opts.SkipTests = append(opts.SkipTests, skippedTestsForTLSPassthrough...)
	opts.SkipTests = append(opts.SkipTests, skippedTestsForKnownGaps...)
	// Implementation is left to the flags DefaultOptions already applied.
	// Assigning it here would override them and pin the report to a stale version.

	conformance.RunConformanceWithOptions(t, opts)
}
