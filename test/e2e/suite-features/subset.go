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
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-features: service subset", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("subset not found", func() {
			assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(2), "scaling number of httpbin instances")
			assert.Nil(ginkgo.GinkgoT(), s.WaitAllHTTPBINPodsAvailable(), "waiting for all httpbin pods ready")
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
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
      - httpbin.com
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
      subset: not_exist
`, backendSvc, backendSvcPort[0])
			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ApisixRoute")

			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 0, "upstreams nodes not expect")

			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusServiceUnavailable).Body().Raw()
		})

		ginkgo.It("subset with bad labels", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: %s
spec:
  subsets:
  - name: aa
    labels:
      aa: bb
      cc: dd
`, backendSvc)
			err := s.CreateResourceFromString(au)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")
			time.Sleep(1 * time.Second)
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
      - httpbin.com
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
      subset: aa
`, backendSvc, backendSvcPort[0])
			err = s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ApisixRoute")
			time.Sleep(3 * time.Second)
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 0, "upstreams nodes not expect")

			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusServiceUnavailable).Body().Raw()
		})

		ginkgo.It("subset with good labels - all", func() {
			assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(2), "scaling number of httpbin instances")
			assert.Nil(ginkgo.GinkgoT(), s.WaitAllHTTPBINPodsAvailable(), "waiting for all httpbin pods ready")

			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: %s
spec:
  subsets:
  - name: all
    labels:
      app: httpbin-deployment-e2e-test
`, backendSvc)
			err := s.CreateResourceFromString(au)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")
			time.Sleep(1 * time.Second)
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
      - httpbin.com
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
      subset: all
`, backendSvc, backendSvcPort[0])
			err = s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ApisixRoute")

			time.Sleep(6 * time.Second)
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2, "upstreams nodes not expect")

			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
		})
	}

	ginkgo.Describe("suite-features: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold)
	})
	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
