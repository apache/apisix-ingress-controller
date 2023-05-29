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
	"net/http"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress-features: CRDs status subresource Testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("check ApisixRoute status is recorded", func() {
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
		time.Sleep(6 * time.Second)
		// status should be recorded as successful
		output, err := s.GetOutputFromString("ar", "httpbin-route", "-o", "yaml")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ApisixRoute resource")
		assert.Contains(ginkgo.GinkgoT(), output, "type: ResourcesAvailable", "status.conditions.type is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "reason: ResourcesSynced", "status.conditions.reason  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, `status: "True"`, "status.conditions.status  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "message: Sync Successfully", "status.conditions.message  is recorded")

		apisixRoute = fmt.Sprintf(`
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
    plugins:
    - name: non-existent
      enable: true
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))
		time.Sleep(6 * time.Second)
		// status should be recorded as successful
		output, err = s.GetOutputFromString("ar", "httpbin-route", "-o", "yaml")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ApisixRoute resource")
		assert.Contains(ginkgo.GinkgoT(), output, "type: ResourcesAvailable", "status.conditions.type is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "reason: ResourceSyncAborted", "status.conditions.reason  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, `status: "False"`, "status.conditions.status  is recorded")
	})

	ginkgo.It("check the ApisixUpstream status is recorded", func() {
		backendSvc, _ := s.DefaultHTTPBackend()
		apisixUpstream := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  retries: 2
`, backendSvc)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixUpstream))

		// status should be recorded as successful
		output, err := s.GetOutputFromString("au", backendSvc, "-o", "yaml")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ApisixRoute resource")
		assert.Contains(ginkgo.GinkgoT(), output, "type: ResourcesAvailable", "status.conditions.type is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "reason: ResourceSyncAborted", "status.conditions.reason  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, `status: "False"`, "status.conditions.status  is recorded")
	})

	ginkgo.It("check ApisixPluginConfig status is recorded", func() {
		apc := `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: test-apc
spec:
 plugins:
 - name: echo
   enable: true
   config:
    body: "my custom body"
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))
		time.Sleep(6 * time.Second)
		// status should be recorded as successful
		output, err := s.GetOutputFromString("apc", "test-apc", "-o", "yaml")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ApisixPluginConfig resource")
		assert.Contains(ginkgo.GinkgoT(), output, "type: ResourcesAvailable", "status.conditions.type is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "reason: ResourcesSynced", "status.conditions.reason  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, `status: "True"`, "status.conditions.status  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "message: Sync Successfully", "status.conditions.message  is recorded")

		apc = `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: test-apc
spec:
 plugins:
 - name: echo
   enable: true
   config:
    body: "my custom body"
 - name: non-existent
   enable: true
`

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))
		time.Sleep(6 * time.Second)
		// status should be recorded as failed
		output, err = s.GetOutputFromString("apc", "test-apc", "-o", "yaml")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ApisixPluginConfig resource")
		assert.Contains(ginkgo.GinkgoT(), output, "type: ResourcesAvailable", "status.conditions.type is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "reason: ResourceSyncAborted", "status.conditions.reason  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, `status: "False"`, "status.conditions.status  is recorded")
	})

	ginkgo.It("check ApisixTls status is recorded", func() {
		secretName := "test-apisix-tls"
		cert := `-----BEGIN CERTIFICATE-----
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
		key := `-----BEGIN RSA PRIVATE KEY-----
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
		// create secret
		err := s.NewSecret(secretName, cert, key)
		assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
		// create ApisixTls resource
		tlsName := "tls-name"
		host := "api6.com"
		assert.Nil(ginkgo.GinkgoT(), s.NewApisixTls(tlsName, host, secretName), "create tls error")
		time.Sleep(6 * time.Second)
		// status should be recorded as successful
		output, err := s.GetOutputFromString("atls", tlsName, "-o", "yaml")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ApisixPluginConfig resource")
		assert.Contains(ginkgo.GinkgoT(), output, "type: ResourcesAvailable", "status.conditions.type is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "reason: ResourcesSynced", "status.conditions.reason  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, `status: "True"`, "status.conditions.status  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "message: Sync Successfully", "status.conditions.message  is recorded")

		// No secret exists for use
		assert.Nil(ginkgo.GinkgoT(), s.NewApisixTls(tlsName, host, "non-existent.com"), "create tls error")
		time.Sleep(6 * time.Second)
		// status should be recorded as failed
		output, err = s.GetOutputFromString("atls", tlsName, "-o", "yaml")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ApisixPluginConfig resource")
		assert.Contains(ginkgo.GinkgoT(), output, "type: ResourcesAvailable", "status.conditions.type is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "reason: ResourceSyncAborted", "status.conditions.reason  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, `status: "False"`, "status.conditions.status  is recorded")
	})

	//TODO: ApisixGlobal
	ginkgo.It("check ApisixConsumer status is recorded", func() {
		// create ApisixConsumer resource
		assert.Nil(ginkgo.GinkgoT(), s.ApisixConsumerBasicAuthCreated("test-apisix-consumer", "foo", "bar"), "create consumer error")
		time.Sleep(6 * time.Second)

		// status should be recorded as successful
		output, err := s.GetOutputFromString("ac", "test-apisix-consumer", "-o", "yaml")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ApisixConsumer resource")
		assert.Contains(ginkgo.GinkgoT(), output, "type: ResourcesAvailable", "status.conditions.type is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "reason: ResourcesSynced", "status.conditions.reason  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, `status: "True"`, "status.conditions.status  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "message: Sync Successfully", "status.conditions.message  is recorded")
	})
})

var _ = ginkgo.Describe("suite-ingress-features: Ingress LB Status Testing", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		IngressAPISIXReplicas: 1,
		APISIXPublishAddress:  "10.6.6.6",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("check the ingress lb status is updated", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-v1-lb
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)

		output, err := s.GetOutputFromString("ingress", "ingress-v1-lb", "-o", "jsonpath='{ .status.loadBalancer.ingress[0].ip }'")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ingress status")

		hasIP := strings.Contains(output, "10.6.6.6")
		assert.True(ginkgo.GinkgoT(), hasIP, "LB Status is recorded")
	})
})

var _ = ginkgo.Describe("suite-ingress-features: disable status", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		IngressAPISIXReplicas: 1,
		APISIXPublishAddress:  "10.6.6.6",
		DisableStatus:         true,
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("check the ApisixRoute status is recorded", func() {
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
		// status should be recorded as successful
		output, err := s.GetOutputFromString("ar", "httpbin-route", "-o", "jsonpath='{ .status }'")
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Equal(ginkgo.GinkgoT(), "''", output)
	})

	ginkgo.It("check the ingress lb status is updated", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-v1-lb
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)

		output, err := s.GetOutputFromString("ingress", "ingress-v1-lb", "-o", "jsonpath='{ .status.loadBalancer }'")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ingress status")

		assert.Equal(ginkgo.GinkgoT(), "'{}'", output)
	})
})
