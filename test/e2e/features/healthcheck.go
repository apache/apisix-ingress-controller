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
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("health check", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("active check", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()

		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: %s
spec:
  healthCheck:
    active:
      type: http
      httpPath: /status/502
      healthy:
        httpCodes: [200]
        interval: 1s
      unhealthy:
        httpFailures: 2
        interval: 1s
`, backendSvc)
		err := s.CreateResourceFromString(au)
		assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")

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
      - /*
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendPorts[0])
		err = s.CreateResourceFromString(ar)
		assert.Nil(ginkgo.GinkgoT(), err)

		time.Sleep(3 * time.Second)

		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, nil, "listing upstreams")
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Checks.Active.Healthy.Interval, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Checks.Active.Healthy.HTTPStatuses, []int{200})
		assert.Equal(ginkgo.GinkgoT(), ups[0].Checks.Active.Unhealthy.Interval, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Checks.Active.Unhealthy.HTTPFailures, 2)

		// It's difficult to test healthchecker since we cannot let partial httpbin endpoints
		// down, if all of them are down, apisix in turn uses all of them.
	})

	ginkgo.It("passive check", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()

		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: %s
spec:
  healthCheck:
    active:
      type: http
      httpPath: /status/200
      healthy:
        httpCodes: [200]
        interval: 1s
      unhealthy:
        httpFailures: 2
        interval: 1s
    passive:
      healthy:
        httpCodes: [200]
      unhealthy:
        httpCodes: [502]
`, backendSvc)
		err := s.CreateResourceFromString(au)
		assert.Nil(ginkgo.GinkgoT(), err, "create ApisixUpstream")

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
      - /*
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendPorts[0])
		err = s.CreateResourceFromString(ar)
		assert.Nil(ginkgo.GinkgoT(), err)

		time.Sleep(3 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, nil, "listing upstreams")
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Checks.Active.Healthy.Interval, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Checks.Active.Healthy.HTTPStatuses, []int{200})
		assert.Equal(ginkgo.GinkgoT(), ups[0].Checks.Active.Unhealthy.Interval, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Checks.Passive.Healthy.HTTPStatuses, []int{200})
		assert.Equal(ginkgo.GinkgoT(), ups[0].Checks.Passive.Unhealthy.HTTPStatuses, []int{502})
	})
})
