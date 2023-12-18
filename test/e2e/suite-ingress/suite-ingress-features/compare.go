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
package ingress

import (
	"fmt"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress-features: Testing compare resources", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("Compare and find out the redundant objects in APISIX, and remove them", func() {
			if s.IsEtcdServer() {
				ginkgo.Skip("Does not support etcdserver mode, etcdserver does not support full synchronization")
			}
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			// scale Ingres Controller --replicas=0
			assert.Nil(ginkgo.GinkgoT(), s.ScaleIngressController(0), "scaling ingress controller instances = 0")
			// remove ApisixRoute resource
			assert.Nil(ginkgo.GinkgoT(), s.RemoveResourceByString(apisixRoute))
			// scale Ingres Controller --replicas=1
			assert.Nil(ginkgo.GinkgoT(), s.ScaleIngressController(1), "scaling ingress controller instances = 1")
			time.Sleep(15 * time.Second)
			// should find the warn log
			output := s.GetDeploymentLogs("ingress-apisix-controller-deployment-e2e-test")

			assert.Contains(ginkgo.GinkgoT(), output, "in APISIX but do not in declare yaml")
		})
	}

	ginkgo.Describe("suite-ingress-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
