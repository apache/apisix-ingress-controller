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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/gateway-api/conformance"
	conformancev1 "sigs.k8s.io/gateway-api/conformance/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/tests"
	"sigs.k8s.io/gateway-api/conformance/utils/flags"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/yaml"
)

// https://github.com/kubernetes-sigs/gateway-api/blob/5c5fc388829d24e8071071b01e8313ada8f15d9f/conformance/utils/suite/suite.go#L358.  SAN includes '*'
var skippedTestsForSSL = []string{
	tests.HTTPRouteHTTPSListener.ShortName,
	tests.HTTPRouteRedirectPortAndScheme.ShortName,
}

// TODO: HTTPRoute hostname intersection and listener hostname matching

func TestGatewayAPIConformance(t *testing.T) {
	opts := conformance.DefaultOptions(t)
	opts.Debug = true
	opts.CleanupBaseResources = true
	opts.GatewayClassName = gatewayClassName
	opts.SkipTests = skippedTestsForSSL
	opts.Implementation = conformancev1.Implementation{
		Organization: "APISIX",
		Project:      "apisix-ingress-controller",
		URL:          "https://github.com/apache/apisix-ingress-controller.git",
		Version:      "v2.0.0",
	}

	cSuite, err := suite.NewConformanceTestSuite(opts)
	require.NoError(t, err)

	t.Log("starting the gateway conformance test suite")
	cSuite.Setup(t, tests.ConformanceTests)

	if err := cSuite.Run(t, tests.ConformanceTests); err != nil {
		t.Fatalf("failed to run the gateway conformance test suite: %v", err)
	}

	report, err := cSuite.Report()
	if err != nil {
		t.Fatalf("failed to get the gateway conformance test report: %v", err)
	}

	rawReport, err := yaml.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal the gateway conformance test report: %v", err)
	}
	f, err := os.Create(*flags.ReportOutput)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	_, err = f.Write(rawReport)
	require.NoError(t, err)
}
