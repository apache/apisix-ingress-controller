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

package webhook

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test TCPRoute Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "tcproute-webhook-test",
		EnableWebhook: true,
	})

	BeforeEach(func() {
		setupSimpleGatewayWithProtocol(s, "TCP", "tcp", 9000)
	})

	It("should warn on missing backend services", func() {
		tc := simpleRouteWebhookTestCase{
			routeKind:       "TCPRoute",
			routeName:       "webhook-tcproute",
			sectionName:     "tcp",
			missingService:  "missing-tcp-backend",
			servicePortName: "tcp",
			servicePort:     80,
			serviceProtocol: "",
		}
		verifySimpleRouteMissingBackendWarnings(s, tc)
	})
})
