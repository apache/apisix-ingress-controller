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
		assert.Equal(ginkgo.GinkgoT(), *ups[0].Retries, 3)

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
		assert.Equal(ginkgo.GinkgoT(), *ups[0].Retries, 2)

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("ApisixGlobalRule should be ignored", func() {
		agr := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-1
spec:
  ingressClassName: ignore
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world!!"
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "creating ApisixGlobalRule")
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		assert.Len(ginkgo.GinkgoT(), grs, 0)

		s.NewAPISIXClient().GET("/anything").Expect().Body().NotContains("hello, world!!")
		s.NewAPISIXClient().GET("/hello").Expect().Body().NotContains("hello, world!!")
	})

	ginkgo.It("ApisixGlobalRule should be handled", func() {
		agr := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-1
spec:
  ingressClassName: apisix
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world!!"
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "creating ApisixGlobalRule")
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		_, ok := grs[0].Plugins["echo"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		s.NewAPISIXClient().GET("/anything").Expect().Body().Contains("hello, world!!")

		s.NewAPISIXClient().GET("/hello").Expect().Body().Contains("hello, world!!")
	})

	ginkgo.It("ApisixGlobalRule should be without ingressClass", func() {
		agr := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-1
spec:
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world!!"
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "creating ApisixGlobalRule")
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		_, ok := grs[0].Plugins["echo"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		s.NewAPISIXClient().GET("/anything").Expect().Body().Contains("hello, world!!")

		s.NewAPISIXClient().GET("/hello").Expect().Body().Contains("hello, world!!")
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
		assert.Equal(ginkgo.GinkgoT(), *ups[0].Retries, 3)

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
		assert.Equal(ginkgo.GinkgoT(), *ups[0].Retries, 2)

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
		assert.Equal(ginkgo.GinkgoT(), *ups[0].Retries, 1)

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("ApisixGlobalRule should be handled", func() {
		agr := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-1
spec:
  ingressClassName: apisix
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world!!"
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "creating ApisixGlobalRule")
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		_, ok := grs[0].Plugins["echo"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		s.NewAPISIXClient().GET("/anything").Expect().Body().Contains("hello, world!!")

		s.NewAPISIXClient().GET("/hello").Expect().Body().Contains("hello, world!!")
	})

	ginkgo.It("ApisixGlobalRule should be without ingressClass", func() {
		agr := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-1
spec:
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world!!"
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "creating ApisixGlobalRule")
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		_, ok := grs[0].Plugins["echo"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		s.NewAPISIXClient().GET("/anything").Expect().Body().Contains("hello, world!!")

		s.NewAPISIXClient().GET("/hello").Expect().Body().Contains("hello, world!!")
	})

})
