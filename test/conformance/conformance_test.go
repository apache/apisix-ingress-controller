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

	// TODO: APISIX does not support TLSRoute passthrough.
	tests.TLSRouteSimpleSameNamespace.ShortName,
}

// TODO: HTTPRoute hostname intersection and listener hostname matching

func TestGatewayAPIConformance(t *testing.T) {
	opts := conformance.DefaultOptions(t)
	opts.Debug = true
	opts.CleanupBaseResources = true
	opts.GatewayClassName = gatewayClassName
	opts.SkipTests = append(opts.SkipTests, skippedTestsForSSL...)
	opts.Implementation = conformancev1.Implementation{
		Organization: "APISIX",
		Project:      "apisix-ingress-controller",
		URL:          "https://github.com/apache/apisix-ingress-controller.git",
		Version:      "v2.0.0",
	}

	conformance.RunConformanceWithOptions(t, opts)
}
