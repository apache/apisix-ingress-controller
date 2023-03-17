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

const (
	_secretName = "test-apisix-tls"
	_cert       = `-----BEGIN CERTIFICATE-----
MIIDSjCCAjICCQC/34ZwGz7ZXjANBgkqhkiG9w0BAQsFADBmMQswCQYDVQQGEwJD
TjEQMA4GA1UECAwHSmlhbmdzdTEPMA0GA1UEBwwGU3V6aG91MQ8wDQYDVQQKDAZ6
aGlsaXUxEDAOBgNVBAsMB3NlY3Rpb24xETAPBgNVBAMMCHRlc3QuY29tMCAXDTIx
MDIwMzE0MjkwOVoYDzIwNTEwMTI3MTQyOTA5WjBmMQswCQYDVQQGEwJDTjEQMA4G
A1UECAwHSmlhbmdzdTEPMA0GA1UEBwwGU3V6aG91MQ8wDQYDVQQKDAZ6aGlsaXUx
EDAOBgNVBAsMB3NlY3Rpb24xETAPBgNVBAMMCHRlc3QuY29tMIIBIjANBgkqhkiG
9w0BAQEFAAOCAQ8AMIIBCgKCAQEA3DEQ5K9PVYicINTHt3arqrsrftrhotyBuGqM
xxqGMVO/E2SAa/81fC1UCcjYV4Wila0kl8i5fa8HjtVm5UWlrqxeFLOS3E0Wv2QY
w46BGZJY4InE9zKwYyC2DkBxE6p14JRjmtW/MQPNaOFjJ4bmCuRHsEzmQIGRM0b7
oKHjfFwv6l7BahgGf9ShHOMdHSkgWj6+2RU3282lrO9bY1JBTKu2Znv9M79nu1Px
Tn1wCfcuCwA7WQT/QSrE2R43I2vmbIbuSmeg9ivjMazRYQQ+qxQn/6zhiHvP3QZG
dKmp8imdYi+r84PKOLDEe/yxlgIdr2Au5WCPWwyYMYPWHzeD1wIDAQABMA0GCSqG
SIb3DQEBCwUAA4IBAQBYzNe83mPVuz96TZ3fmxtOIuz9b6q5JWiJiOzjAD9902Se
TNYzMM6T/5e0dBpj8Z2qQlhkfNxJJgTwGEE8SdrZIr8DhswR9a0bXDCZjLatCdeU
iYpt+TDAuySnLhAcd3GfE5ml6am2dOsOKpxHU/8clUSaz+21fckRopWo+xL6rSVC
4vvKqiU+LWLTZPQNoOqowl7bxoQO2jMWfN/5zvQOFxAbEufIPa9ti3qonDCXbkYn
PpET/mPDrcb4bGsZkW/cu0LrPSUVp12br5TAYaXqYS0Ex+jAVTXML9SeEQuvU3dH
5Uw2wVHxQXHglsdCYUXXFd3HZffb4rSQH+Mk0CBI
-----END CERTIFICATE-----`
	_key = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEA3DEQ5K9PVYicINTHt3arqrsrftrhotyBuGqMxxqGMVO/E2SA
a/81fC1UCcjYV4Wila0kl8i5fa8HjtVm5UWlrqxeFLOS3E0Wv2QYw46BGZJY4InE
9zKwYyC2DkBxE6p14JRjmtW/MQPNaOFjJ4bmCuRHsEzmQIGRM0b7oKHjfFwv6l7B
ahgGf9ShHOMdHSkgWj6+2RU3282lrO9bY1JBTKu2Znv9M79nu1PxTn1wCfcuCwA7
WQT/QSrE2R43I2vmbIbuSmeg9ivjMazRYQQ+qxQn/6zhiHvP3QZGdKmp8imdYi+r
84PKOLDEe/yxlgIdr2Au5WCPWwyYMYPWHzeD1wIDAQABAoIBAQDGmSKpgE1H0k0v
d3siyFART3vtkLHOWKBPmxqaQhwixWwjq5QA1FCDTcbshFBMsGVyJpZIqGxVJdbl
RyjlRaooH6NDfKvLM2R+/2Mujot2av7qlpgmdXuODOTnecwDds2W33/vGTa2mL1e
CVuLPSqjTD40j0dlivdRjoZJ3Xn2oOrpZ812XU8KeZAjuSEAwcyl2nSbyLGDchBB
kfYZold3FaaLAf2LoVJ2fs+FwEPzDKoNYEvij9OyC0kwI94T5jQ+Z6XGtHXhb2Hy
Ek3EZeIhV3YcDIid5AjSvcrNtDI24hwszSmhYVc53EKYkpXHf581a3U/SEEhXDlw
Y0x6j9QRAoGBAPEP0LDgP7DGXxno4h+nf0AMru0pxlrNVQhLcNQB+dYI0pFTwsg+
AKenoaaE/EGR1KfiY0uf3rVWNrA5kyX1/i18iJx9LSf9NvNgMo84JVvXINgyE6sd
hvdqxFlV5FBnh8b7ldvYQy3YI0EQNx+/rmeUYPjInbkdiksAtAey4ADNAoGBAOnW
K0FoX1ljq3rc9uVasiRi6Ix50NHdZ17RcEpMgwWPenbP1aiWkvA8yFhU708lBaZC
WIUZ6XbfiG0Y9cMtxhtikoouDs5Ifia8juZ2bhkmSGP2FvZCBJJ/sHhnhpzSZNhW
SyLBUjnynoXwHoQvkoGnVTHAk1VsY7jLNJdr2MczAoGAMYvMmu4caRr8pPimsVbd
4q44reouKK+XUJMg55JYZVN+4/vRRxLnU44yvWUL6/YrPS5ctkhvn9nOd739rom2
6mZ0NaXMyDFVQAR/n8wscYnv6D+ypzL0cJnzLWFoAdalo5JGJN94P03zQQYyLkZZ
dFSc8cVaFZgqumu0lPiA7ekCgYEAiMeVL8Jcm84YXVrpNMmzkGMmwhzzT/8hWy5J
b7yHm3YM3Xi+8sl5E/uJ+VldTj9KqbD/VIQOs1EX3TEPeObKjfQ/4YIFeRagbAo5
0IcP6bgh+g7V6aA+Sm9Ui2mLLSpIgN8hPig0796CabhGMW4eVabKx7pstDgdsNd0
YOpduE8CgYEAu9k9WOQuRX4f6i5LBIxyaYn6Hw6oJn8e/w+p2+HNBXdyVQNqUHBG
V5rgnBwhc5LeIFbehKvQOvYSWbwbA1VunMpdYgV6+EBLayumJNqV6jGei4okx2of
wrw7im4TNSAdwVX4Y1F4svJ2as5SJn5QYGAzXDixNuwzXYrpP9rzA2s=
-----END RSA PRIVATE KEY-----`
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

	ginkgo.It("ApisixTls should be handled", func() {
		err := s.NewSecret(_secretName, _cert, _key)
		assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
		// create ApisixTls resource without ingressClassName
		tlsName := "tls-name"
		host := "api6.com"
		err = s.NewApisixTls(tlsName, host, _secretName)
		assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
		time.Sleep(6 * time.Second)

		// check ssl in APISIX
		tls, err := s.ListApisixSsl()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
		assert.Equal(ginkgo.GinkgoT(), tls[0].Snis[0], host, "tls host is error")

		// update ApisixTls resource with ingressClassName: apisix
		host2 := "api7.com"
		err = s.NewApisixTls(tlsName, host2, _secretName, "apisix")
		assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
		time.Sleep(6 * time.Second)

		// check ssl in APISIX
		tls, err = s.ListApisixSsl()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
		assert.Equal(ginkgo.GinkgoT(), tls[0].Snis[0], host2, "tls host is error")
	})

	ginkgo.It("ApisixTls should be ignored", func() {
		err := s.NewSecret(_secretName, _cert, _key)
		assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
		// create ApisixTls resource with ingressClassName: ignored
		tlsName := "tls-name"
		host := "api6.com"
		err = s.NewApisixTls(tlsName, host, _secretName, "ignored")
		assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
		time.Sleep(6 * time.Second)
		// check ssl in APISIX
		tls, err := s.ListApisixSsl()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 0, "tls number not expect")
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

	ginkgo.It("ApisixClusterConfig should be ignored", func() {
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

	ginkgo.It("ApisixClusterConfig should be handled", func() {
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

	ginkgo.It("ApisixRoute should be ignored", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()

		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: test-ar-1
spec:
  ingressClassName: ignore
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

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		// The referenced plugin doesn't exist so the translation expected to be failed
		err := s.EnsureNumApisixUpstreamsCreated(0)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(0)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
	})

	ginkgo.It("ApisixRoute should be handled", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()

		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: test-ar-1
spec:
 http:
 ingressClassName: apisix
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
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("ApisixRoute should be handled without ingressClass", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: test-ar-1
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
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
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

	ginkgo.It("ApisixTls should be handled", func() {
		err := s.NewSecret(_secretName, _cert, _key)
		assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
		// create ApisixTls resource without ingressClassName
		tlsName := "tls-name"
		host := "api6.com"
		err = s.NewApisixTls(tlsName, host, _secretName)
		assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
		time.Sleep(6 * time.Second)
		// check ssl in APISIX
		tls, err := s.ListApisixSsl()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
		assert.Equal(ginkgo.GinkgoT(), tls[0].Snis[0], host, "tls host is error")

		// update ApisixTls resource with ingressClassName: apisix
		host2 := "api7.com"
		err = s.NewApisixTls(tlsName, host2, _secretName, "apisix")
		assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
		time.Sleep(6 * time.Second)
		// check ssl in APISIX
		tls, err = s.ListApisixSsl()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
		assert.Equal(ginkgo.GinkgoT(), tls[0].Snis[0], host2, "tls host is error")

		// update ApisixTls resource with ingressClassName: watch
		host3 := "api7.org"
		err = s.NewApisixTls(tlsName, host3, _secretName, "watch")
		assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
		time.Sleep(6 * time.Second)
		// check ssl in APISIX
		tls, err = s.ListApisixSsl()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
		assert.Equal(ginkgo.GinkgoT(), tls[0].Snis[0], host3, "tls host is error")
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

	ginkgo.It("ApisixClusterConfig should be handled", func() {
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

	ginkgo.It("ApisixRoute should be handled", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()

		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: test-ar-1
spec:
 http:
 ingressClassName: apisix
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
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("ApisixRoute should be handled without ingressClass", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()

		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: test-ar-1
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
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})
})
