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
package endpoints

import (
	"fmt"
	"net/http"
	"time"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("endpoints", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("ignore applied only if there is an ApisixRoute referenced", func() {
		time.Sleep(5 * time.Second)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(0), "checking number of upstreams")
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		ups := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
   rules:
   - host: httpbin.org
     http:
       paths:
       - backend:
           serviceName: %s
           servicePort: %d
         path: /ip
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ups))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")
	})

	ginkgo.It("upstream nodes should be reset to empty when Service/Endpoints was deleted", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 rules:
 - host: httpbin.com
   http:
     paths:
     - backend:
         serviceName: %s
         servicePort: %d
       path: /ip
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

		// Now delete the backend httpbin service resource.
		assert.Nil(ginkgo.GinkgoT(), s.DeleteHTTPBINService())
		time.Sleep(3 * time.Second)
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusServiceUnavailable)
	})
})
