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

	"github.com/apache/apisix-ingress-controller/pkg/id"
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
	ginkgo.It("ApisixUpstream should be ignored", func() {
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

	ginkgo.It("ApisixUpstream should be handled", func() {
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

	ginkgo.It("ApisixPluginConfig should be ignored", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-apc-1
spec:
  ingressClassName: ignored
  plugins:
  - name: echo
    enable: true
    config:
      body: "my custom body"
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		ar := fmt.Sprintf(`
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
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		// The referenced plugin doesn't exist so the translation expected to be failed
		err := s.EnsureNumApisixUpstreamsCreated(0)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(0)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(0)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
	})

	ginkgo.It("ApisixPluginConfig should be handled", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-apc-1
spec:
  ingressClassName: apisix
  plugins:
  - name: echo
    enable: true
    config:
      body: "my custom body"
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		ar := fmt.Sprintf(`
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
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Body().IsEqual("my custom body")
	})

	ginkgo.It("ApisixPluginConfig should be handled without ingressClass", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-apc-1
spec:
  plugins:
  - name: echo
    enable: true
    config:
      body: "my custom body"
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		ar := fmt.Sprintf(`
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
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Body().IsEqual("my custom body")
	})

	ginkgo.It("ApisiClusterConfig should be ignored", func() {
		// create ApisixConsumer resource with ingressClassName: ignore
		acc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: ignore
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok := agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
	})

	ginkgo.It("ApisiClusterConfig should be handled", func() {
		// create ApisixConsumer resource without ingressClassName
		acc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok := agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		// update ApisixConsumer resource with ingressClassName: apisix
		acc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: apisix
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
	})
})

var _ = ginkgo.Describe("suite-ingress-features: Testing CRDs with IngressClass apisix-and-all", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "ingress-class",
		IngressAPISIXReplicas: 1,
		IngressClass:          "apisix-and-all",
	})

	ginkgo.It("ApisixUpstream should be handled", func() {
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

	ginkgo.It("ApisixPluginConfig should be handled", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-apc-1
spec:
  ingressClassName: apisix
  plugins:
  - name: echo
    enable: true
    config:
      body: "my custom body"
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		ar := fmt.Sprintf(`
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
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Body().IsEqual("my custom body")
	})

	ginkgo.It("ApisixPluginConfig should be handled without ingressClass", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-apc-1
spec:
  plugins:
  - name: echo
    enable: true
    config:
      body: "my custom body"
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		ar := fmt.Sprintf(`
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
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Body().IsEqual("my custom body")
	})

	ginkgo.It("ApisiClusterConfig should be handled", func() {
		// create ApisixConsumer resource without ingressClassName
		acc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok := agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		// update ApisixConsumer resource with ingressClassName: apisix
		acc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: apisix
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		// update ApisixConsumer resource with ingressClassName: watch
		acc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: watch
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
	})
})
