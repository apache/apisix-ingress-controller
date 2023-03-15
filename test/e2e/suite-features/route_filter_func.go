// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package features

import (
	"fmt"
	"net/http"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-features: filter_func", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("filter using body", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
       - /ip
     filter_func: "function(vars)\n  local core = require ('apisix.core')\n  local body, err = core.request.get_body()\n  if not body then\n      return false\n  end\n\n  local data, err = core.json.decode(body)\n  if not data then\n      return false\n  end\n\n  if data['foo'] == 'bar' then\n      return true\n  end\n\n  return false\nend"
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Contains("404 Route Not Found")

			type MyJSON struct {
				Foo string `json:"foo"`
			}

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithJSON(MyJSON{Foo: "bar"}).
				Expect().
				Status(http.StatusOK)
		})
	}

	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
