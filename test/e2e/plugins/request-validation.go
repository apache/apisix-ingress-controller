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
package plugins

import (
	"fmt"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("redirect plugin", func() {
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
       - /post
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: request-validation
     enable: true
     config:
       body_schema:
         type: object
         properties:
           name:
             type: string
             minLength: 5
           id:
             type: integer
             minimum: 20
         required:
         - id
       header_schema:
         type: object
         properties:
           user-agent:
             type: string
             pattern: .*Mozilla.*
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		// header schema check failure.
		resp := s.NewAPISIXClient().POST("/post").WithHeader("Host", "httpbin.org").WithHeader("User-Agent", "bad-ua").Expect()
		resp.Status(400)

		payload := []byte(`
{
    "name": "bob",
    "id": 33
}
		`)
		// body schema check failure.
		resp = s.NewAPISIXClient().POST("/post").WithHeader("Host", "httpbin.org").WithHeader("User-Agent", "aaaMozillabb").WithBytes(payload).Expect()
		resp.Status(400)

		payload = []byte(`
{
    "name": "long-name",
    "id": 11
}
		`)
		// body schema check failure.
		resp = s.NewAPISIXClient().POST("/post").WithHeader("Host", "httpbin.org").WithHeader("User-Agent", "aaaMozillabb").WithBytes(payload).Expect()
		resp.Status(400)

		payload = []byte(`
{
    "name": "long-name",
    "id": 55
}
		`)
		resp = s.NewAPISIXClient().POST("/post").WithHeader("Host", "httpbin.org").WithHeader("User-Agent", "aaaMozillabb").WithBytes(payload).Expect()
		resp.Status(200)
	})

	ginkgo.It("disable plugin", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
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
       - /post
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: request-validation
     enable: false
     config:
       body_schema:
         type: object
         properties:
           name:
             type: string
             minLength: 5
           id:
             type: integer
             minimum: 20
         required:
         - id
       header_schema:
         type: object
         properties:
           user-agent:
             type: string
             pattern: .*Mozilla.*
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		resp := s.NewAPISIXClient().POST("/post").WithHeader("Host", "httpbin.org").WithHeader("User-Agent", "bad-ua").Expect()
		resp.Status(200)

		payload := []byte(`
{
    "name": "bob",
    "id": 33
}
		`)
		// body schema check failure.
		resp = s.NewAPISIXClient().POST("/post").WithHeader("Host", "httpbin.org").WithHeader("User-Agent", "aaaMozillabb").WithBytes(payload).Expect()
		resp.Status(200)

		payload = []byte(`
{
    "name": "long-name",
    "id": 11
}
		`)
		// body schema check failure.
		resp = s.NewAPISIXClient().POST("/post").WithHeader("Host", "httpbin.org").WithHeader("User-Agent", "aaaMozillabb").WithBytes(payload).Expect()
		resp.Status(200)

		payload = []byte(`
{
    "name": "long-name",
    "id": 55
}
		`)
		resp = s.NewAPISIXClient().POST("/post").WithHeader("Host", "httpbin.org").WithHeader("User-Agent", "aaaMozillabb").WithBytes(payload).Expect()
		resp.Status(200)
	})
})
