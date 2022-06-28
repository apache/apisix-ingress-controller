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

package ingress

import (
	"fmt"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress: Enable webhooks", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("should fail to create the ApisixRoute with invalid plugin configuration", func() {
			// #FIXME: just skip this case and we can enable it on other PR
			ginkgo.Skip("just skip this case")
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /status/*
   backends:
   - serviceName: %s
     servicePort: %d
     resolveGranularity: service
   plugins:
   - name: api-breaker
     enable: true
     config:
       break_response_code: 100 # should in [200, 599]
`, backendSvc, backendPorts[0])

			err := s.CreateResourceFromString(ar)
			assert.Error(ginkgo.GinkgoT(), err, "Failed to create ApisixRoute")
			assert.Contains(ginkgo.GinkgoT(), err.Error(), "admission webhook")
			assert.Contains(ginkgo.GinkgoT(), err.Error(), "denied the request")
			assert.Contains(ginkgo.GinkgoT(), err.Error(), "api-breaker plugin's config is invalid")
			assert.Contains(ginkgo.GinkgoT(), err.Error(), "Must be greater than or equal to 200")
		})
	}

	ginkgo.Describe("suite-ingress: scaffold v2beta3", func() {
		suites(scaffold.NewScaffold(&scaffold.Options{
			Name:                  "leaderelection",
			Kubeconfig:            scaffold.GetKubeconfig(),
			APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
			IngressAPISIXReplicas: 1,
			HTTPBinServicePort:    80,
			ApisixResourceVersion: scaffold.ApisixResourceVersion().V2beta3,
			EnableWebhooks:        false,
		}))
	})
	ginkgo.Describe("suite-ingress: scaffold v2", func() {
		suites(scaffold.NewScaffold(&scaffold.Options{
			Name:                  "leaderelection",
			Kubeconfig:            scaffold.GetKubeconfig(),
			APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
			IngressAPISIXReplicas: 1,
			HTTPBinServicePort:    80,
			ApisixResourceVersion: scaffold.ApisixResourceVersion().V2,
			EnableWebhooks:        false,
		}))
	})
})
