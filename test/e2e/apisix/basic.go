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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("APISIX Standalone Basic Tests", Label("apisix.apache.org", "v2", "basic"), func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "apisix.apache.org/apisix-ingress-controller",
	})

	Describe("APISIX HTTP Proxy", func() {
		It("should handle basic HTTP requests", func() {
			httpClient := s.NewAPISIXClient()
			Expect(httpClient).NotTo(BeNil())

			// Test basic connectivity
			httpClient.GET("/anything").
				Expect().
				Status(404).Body().Contains("404 Route Not Found")
		})

		It("should handle basic HTTP requests with additional gateway", func() {
			additionalGatewayID, _, err := s.Deployer.CreateAdditionalGateway("additional-gw")
			Expect(err).NotTo(HaveOccurred())

			httpClient, err := s.NewAPISIXClientForGateway(additionalGatewayID)
			Expect(err).NotTo(HaveOccurred())
			Expect(httpClient).NotTo(BeNil())

			httpClient.GET("/anything").
				Expect().
				Status(404).Body().Contains("404 Route Not Found")
		})

	})
})
