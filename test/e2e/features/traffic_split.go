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
package features

import (
	"fmt"
	"math"
	"net/http"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("traffic split", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("sanity", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		adminSvc, adminPort := s.ApisixAdminServiceAndPort()
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
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
       - /get
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   - serviceName: %s
     servicePort: %d
     weight: 5
`, backendSvc, backendPorts[0], adminSvc, adminPort)

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

		err := s.EnsureNumApisixUpstreamsCreated(2)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		// Send requests to APISIX.
		var (
			num404 int
			num200 int
		)
		for i := 0; i < 90; i++ {
			// For requests sent to http-admin, 404 will be given.
			// For requests sent to httpbin, 200 will be given.
			resp := s.NewAPISIXClient().GET("/get").WithHeader("Host", "httpbin.org").Expect()
			status := resp.Raw().StatusCode
			if status != http.StatusOK && status != http.StatusNotFound {
				assert.FailNow(ginkgo.GinkgoT(), "invalid status code")
			}
			if status == 200 {
				num200++
				resp.Body().Contains("origin")
			} else {
				num404++
			}
		}
		dev := math.Abs(float64(num200)/float64(num404) - float64(2))
		assert.Less(ginkgo.GinkgoT(), dev, 0.2)
	})

	ginkgo.It("zero-weight", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		adminSvc, adminPort := s.ApisixAdminServiceAndPort()
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
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
       - /get
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 100
   - serviceName: %s
     servicePort: %d
     weight: 0
`, backendSvc, backendPorts[0], adminSvc, adminPort)

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

		err := s.EnsureNumApisixUpstreamsCreated(2)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		// Send requests to APISIX.
		var (
			num404 int
			num200 int
		)
		for i := 0; i < 90; i++ {
			// For requests sent to http-admin, 404 will be given.
			// For requests sent to httpbin, 200 will be given.
			resp := s.NewAPISIXClient().GET("/get").WithHeader("Host", "httpbin.org").Expect()
			status := resp.Raw().StatusCode
			if status != http.StatusOK && status != http.StatusNotFound {
				assert.FailNow(ginkgo.GinkgoT(), "invalid status code")
			}
			if status == 200 {
				num200++
				resp.Body().Contains("origin")
			} else {
				num404++
			}
		}
		assert.Equal(ginkgo.GinkgoT(), num404, 0)
		assert.Equal(ginkgo.GinkgoT(), num200, 90)
	})
})
