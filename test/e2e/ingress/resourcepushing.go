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
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/api7/ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("ApisixRoute Testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("create and then scale upstream pods to 2 ", func() {
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
         serviceName: httpbin-service-e2e-test
         servicePort: 80
       path: /ip
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))

		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		scale := 2
		err = s.ScaleHTTPBIN(scale)
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(10 * time.Second) // wait for ingress to sync
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2, "upstreams nodes not expect")
	})

	ginkgo.It("create and then remove", func() {
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

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

		// remove
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		time.Sleep(5 * time.Second) // wait for ingress to sync
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), len(ups), 0, "upstreams nodes not expect")
	})
})
