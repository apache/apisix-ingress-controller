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

//go:build conformance
// +build conformance

package conformance

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/gateway-api/conformance"
	conformancev1 "sigs.k8s.io/gateway-api/conformance/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/tests"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/pkg/features"
	"sigs.k8s.io/yaml"
)

var skippedTestsForSSL = []string{
	// Reason: https://github.com/kubernetes-sigs/gateway-api/blob/5c5fc388829d24e8071071b01e8313ada8f15d9f/conformance/utils/suite/suite.go#L358.  SAN includes '*'
	tests.HTTPRouteHTTPSListener.ShortName,
	tests.HTTPRouteRedirectPortAndScheme.ShortName,
}

// TODO: HTTPRoute hostname intersection and listener hostname matching

var gatewaySupportedFeatures = []features.FeatureName{
	features.SupportGateway,
	features.SupportHTTPRoute,
	// features.SupportHTTPRouteMethodMatching,
	// features.SupportHTTPRouteResponseHeaderModification,
	// features.SupportHTTPRouteRequestMirror,
	// features.SupportHTTPRouteBackendRequestHeaderModification,
	// features.SupportHTTPRouteHostRewrite,
}

func TestGatewayAPIConformance(t *testing.T) {
	flag.Parse()

	opts := conformance.DefaultOptions(t)
	opts.Debug = true
	opts.CleanupBaseResources = true
	opts.GatewayClassName = gatewayClassName
	opts.SupportedFeatures = sets.New(gatewaySupportedFeatures...)
	opts.SkipTests = skippedTestsForSSL
	opts.Implementation = conformancev1.Implementation{
		Organization: "APISIX",
		Project:      "apisix-ingress-controller",
		URL:          "https://github.com/apache/apisix-ingress-controller.git",
		Version:      "v2.0.0",
	}
	opts.ConformanceProfiles = sets.New(suite.GatewayHTTPConformanceProfileName)

	cSuite, err := suite.NewConformanceTestSuite(opts)
	require.NoError(t, err)

	t.Log("starting the gateway conformance test suite")
	cSuite.Setup(t, tests.ConformanceTests)

	if err := cSuite.Run(t, tests.ConformanceTests); err != nil {
		t.Fatalf("failed to run the gateway conformance test suite: %v", err)
	}

	const reportFileName = "apisix-ingress-controller-conformance-report.yaml"
	report, err := cSuite.Report()
	if err != nil {
		t.Fatalf("failed to get the gateway conformance test report: %v", err)
	}

	rawReport, err := yaml.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal the gateway conformance test report: %v", err)
	}
	// Save report in the root of the repository, file name is in .gitignore.
	require.NoError(t, os.WriteFile("../../../"+reportFileName, rawReport, 0o600))
}
