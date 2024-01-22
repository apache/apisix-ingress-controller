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

var _ = ginkgo.Describe("suite-features: upstream pass host", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		routeTpl := `
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
      - /*
    backends:
    - serviceName: %s
      servicePort: %d
`

		ginkgo.It("is set to node", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(routeTpl, backendSvc, backendPorts[0])
			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(5 * time.Second)

			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  passHost: node
`, backendSvc)
			err = s.CreateVersionedApisixResource(au)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")
			time.Sleep(2 * time.Second)

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Equal(ginkgo.GinkgoT(), "node", ups[0].PassHost)
		})

		ginkgo.It("is set to rewrite with upstream host", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(routeTpl, backendSvc, backendPorts[0])
			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(5 * time.Second)

			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  passHost: rewrite
  upstreamHost: host
`, backendSvc)
			err = s.CreateVersionedApisixResource(au)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")
			time.Sleep(2 * time.Second)

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Equal(ginkgo.GinkgoT(), "rewrite", ups[0].PassHost)
			assert.Equal(ginkgo.GinkgoT(), "host", ups[0].UpstreamHost)
		})

		ginkgo.It("is set to node with upstream host", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(routeTpl, backendSvc, backendPorts[0])
			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(5 * time.Second)

			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  passHost: node
  upstreamHost: host
`, backendSvc)
			err = s.CreateVersionedApisixResource(au)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")
			time.Sleep(2 * time.Second)

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1)
			assert.Equal(ginkgo.GinkgoT(), "node", ups[0].PassHost)
			assert.Equal(ginkgo.GinkgoT(), "host", ups[0].UpstreamHost)
		})

		ginkgo.It("is set to invalid value", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(routeTpl, backendSvc, backendPorts[0])
			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err)
			time.Sleep(5 * time.Second)

			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  passHost: invalid
`, backendSvc)
			err = s.CreateVersionedApisixResource(au)
			assert.NotNil(ginkgo.GinkgoT(), err)
		})
	}

	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
