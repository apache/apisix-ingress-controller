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
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-features: retries", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		routeTpl := `
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
      - /*
    backends:
    - serviceName: %s
      servicePort: %d
`
		ginkgo.It("is missing", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(routeTpl, backendSvc, backendPorts[0])
			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(5 * time.Second)

			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: %s
spec:
`, backendSvc)
			err = s.CreateVersionedApisixResource(au)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")
			time.Sleep(6 * time.Second)

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Nil(ginkgo.GinkgoT(), ups[0].Retries)
		})

		ginkgo.It("is zero", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(routeTpl, backendSvc, backendPorts[0])
			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(5 * time.Second)

			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: %s
spec:
  retries: 0
`, backendSvc)
			err = s.CreateVersionedApisixResource(au)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")
			time.Sleep(2 * time.Second)

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Equal(ginkgo.GinkgoT(), *ups[0].Retries, 0)
		})

		ginkgo.It("is a positive number", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(routeTpl, backendSvc, backendPorts[0])
			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(5 * time.Second)

			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: %s
spec:
  retries: 3
`, backendSvc)
			err = s.CreateVersionedApisixResource(au)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")
			time.Sleep(2 * time.Second)

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Equal(ginkgo.GinkgoT(), *ups[0].Retries, 3)
		})
	}

	ginkgo.Describe("suite-features: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultV2beta3Scaffold)
	})
	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})

var _ = ginkgo.Describe("suite-features: retries timeout", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("retries timeout", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()

			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: %s
spec:
  timeout:
    read: 10s
    send: 10s
`, backendSvc)
			err := s.CreateVersionedApisixResource(au)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")
			time.Sleep(2 * time.Second)

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
      - /*
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendPorts[0])
			err = s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(5 * time.Second)

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Equal(ginkgo.GinkgoT(), ups[0].Timeout.Connect, 60)
			assert.Equal(ginkgo.GinkgoT(), ups[0].Timeout.Read, 10)
			assert.Equal(ginkgo.GinkgoT(), ups[0].Timeout.Send, 10)
		})
	}

	ginkgo.Describe("suite-features: scaffold v2beta3", func() {
		//suites(scaffold.NewDefaultV2beta3Scaffold)
	})
	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
