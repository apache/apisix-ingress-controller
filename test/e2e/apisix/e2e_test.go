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

package apisix

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	_ "github.com/apache/apisix-ingress-controller/test/e2e/crds/v1alpha1"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/crds/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/gatewayapi"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/ingress"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/webhook"
)

// TestAPISIXE2E runs e2e tests using the APISIX standalone mode
func TestAPISIXE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	// init framework
	_ = framework.NewFramework()

	// init newDeployer function
	scaffold.NewDeployer = scaffold.NewAPISIXDeployer

	_, _ = fmt.Fprintf(GinkgoWriter, "Starting APISIX standalone e2e suite\n")
	RunSpecs(t, "apisix standalone e2e suite")
}
