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

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("suite-ingress-resource: ApisixRoute testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("ApisixRoute one rule references 3 services", func() {
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(`
apiVersion: v1
kind: Service
metadata:
  name: httpbin-service-e2e-test1
spec:
  selector:
    app: httpbin-deployment-e2e-test
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 80
  type: ClusterIP

---

apiVersion: v1
kind: Service
metadata:
  name: httpbin-service-e2e-test2
spec:
  selector:
    app: httpbin-deployment-e2e-test
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 80
  type: ClusterIP
`))

		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule
    match:
      hosts:
      - httpbin.org
      paths:
      - /get
    backends:
       - serviceName: %s
         servicePort: %d
       - serviceName: httpbin-service-e2e-test1
         servicePort: 80
       - serviceName: httpbin-service-e2e-test2
         servicePort: 80
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(3))

		assert.Nil(ginkgo.GinkgoT(), s.DeleteResourceFromString(apisixRoute))
		time.Sleep(6 * time.Second)

		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 0)

		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 0)
	})
})
