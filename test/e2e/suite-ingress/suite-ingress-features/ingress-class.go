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
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress-features: Testing CRDs with IngressClass", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "ingress-class",
		IngressAPISIXReplicas: 1,
		IngressClass:          "apisix",
	})
	ginkgo.It("ApisiUpstream should be ignored", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  ingressClassName: ignore
  retries: 3
`, backendSvc)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(au))
		time.Sleep(6 * time.Second)

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
      - httpbin.org
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		time.Sleep(6 * time.Second)

		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Nil(ginkgo.GinkgoT(), ups[0].Retries)

		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("ApisiUpstream should be handled", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  retries: 3
`, backendSvc)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(au))
		time.Sleep(6 * time.Second)

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
      - httpbin.org
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		time.Sleep(6 * time.Second)

		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), 3, *ups[0].Retries)

		au = fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  ingressClassName: apisix
  retries: 2
`, backendSvc)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(au))
		time.Sleep(6 * time.Second)

		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), 2, *ups[0].Retries)

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("ApisixConsumer should be ignored", func() {
		// create ApisixConsumer resource with ingressClassName: ignore
		ac := `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: jack
spec:
  ingressClassName: ignore
  authParameter:
    keyAuth:
      value:
        key: jack-key
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac))
		time.Sleep(6 * time.Second)
		acs, err := s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), acs, 0)

		// update ApisixConsumer resource with ingressClassName: ignore2
		ac = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: jack
spec:
  ingressClassName: ignore2
  authParameter:
    keyAuth:
      value:
        key: jack-key
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac))
		time.Sleep(6 * time.Second)

		acs, err = s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), acs, 0)
	})

	ginkgo.It("ApisixConsumer should be handled", func() {
		// create ApisixConsumer resource withoutput ingressClassName
		ac := `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: jack
spec:
  authParameter:
    keyAuth:
      value:
        key: jack-key
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac))
		time.Sleep(6 * time.Second)

		acs, err := s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), acs, 1)
		assert.Contains(ginkgo.GinkgoT(), acs[0].Username, "jack")
		assert.Equal(ginkgo.GinkgoT(), map[string]interface{}{"key": "jack-key"}, acs[0].Plugins["key-auth"])

		// delete ApisixConsumer
		assert.Nil(ginkgo.GinkgoT(), s.DeleteResourceFromString(ac))
		time.Sleep(6 * time.Second)
		acs, err = s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), acs, 0)

		// create ApisixConsumer resource with ingressClassName: apisix
		ac = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: james
spec:
  ingressClassName: apisix
  authParameter:
    keyAuth:
      value:
        key: james-key
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac))
		time.Sleep(6 * time.Second)

		acs, err = s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), acs, 1)
		assert.Contains(ginkgo.GinkgoT(), acs[0].Username, "james")
		assert.Equal(ginkgo.GinkgoT(), map[string]interface{}{"key": "james-key"}, acs[0].Plugins["key-auth"])
	})
})

var _ = ginkgo.Describe("suite-ingress-features: Testing CRDs with IngressClass apisix-and-all", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "ingress-class",
		IngressAPISIXReplicas: 1,
		IngressClass:          "apisix-and-all",
	})

	ginkgo.It("ApisiUpstream should be handled", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  retries: 3
`, backendSvc)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(au))
		time.Sleep(6 * time.Second)

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
      - httpbin.org
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		time.Sleep(6 * time.Second)

		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), 3, *ups[0].Retries)

		au = fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  ingressClassName: apisix
  retries: 2
`, backendSvc)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(au))
		time.Sleep(6 * time.Second)

		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), 2, *ups[0].Retries)

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)

		au = fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  ingressClassName: watch
  retries: 1
`, backendSvc)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(au))
		time.Sleep(6 * time.Second)

		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), 1, *ups[0].Retries)

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("ApisixConsumer should be handled", func() {
		// create ApisixConsumer resource withoutput ingressClassName
		ac := `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: jack
spec:
  authParameter:
    keyAuth:
      value:
        key: jack-key
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac))
		time.Sleep(6 * time.Second)

		acs, err := s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), acs, 1)
		assert.Contains(ginkgo.GinkgoT(), acs[0].Username, "jack")
		assert.Equal(ginkgo.GinkgoT(), map[string]interface{}{"key": "jack-key"}, acs[0].Plugins["key-auth"])

		// delete ApisixConsumer
		assert.Nil(ginkgo.GinkgoT(), s.DeleteResourceFromString(ac))
		time.Sleep(6 * time.Second)
		acs, err = s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), acs, 0)

		// create ApisixConsumer resource with ingressClassName: apisix
		ac = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: james
spec:
  ingressClassName: apisix
  authParameter:
    keyAuth:
      value:
        key: james-key
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac))
		time.Sleep(6 * time.Second)

		acs, err = s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), acs, 1)
		assert.Contains(ginkgo.GinkgoT(), acs[0].Username, "james")
		assert.Equal(ginkgo.GinkgoT(), map[string]interface{}{"key": "james-key"}, acs[0].Plugins["key-auth"])

		// update ApisixConsumer resource with ingressClassName: watch
		ac = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: james
spec:
  ingressClassName: watch
  authParameter:
    keyAuth:
      value:
        key: james-password
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac))
		time.Sleep(6 * time.Second)

		acs, err = s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), acs, 1)
		assert.Contains(ginkgo.GinkgoT(), acs[0].Username, "james")
		assert.Equal(ginkgo.GinkgoT(), map[string]interface{}{"key": "james-password"}, acs[0].Plugins["key-auth"])
	})
})
