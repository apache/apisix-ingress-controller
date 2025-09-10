// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package v2

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test ApisixRoute With StreamRoute", Label("apisix.apache.org", "v2", "apisixroute"), func() {
	s := scaffold.NewDefaultScaffold()

	BeforeEach(func() {
		if framework.ProviderType != framework.ProviderTypeAPISIX {
			Skip("only support APISIX provider")
		}
		By("create GatewayProxy")
		gatewayProxy := s.GetGatewayProxyYaml()
		err := s.CreateResourceFromString(gatewayProxy)
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	Context("TCP Proxy", func() {
		apisixRoute := `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-tcp-route
spec:
  ingressClassName: %s
  stream:
  - name: rule1
    protocol: TCP
    match:
      ingressPort: 9100
    backend:
      serviceName: httpbin-service-e2e-test
      servicePort: 80
`
		It("stream tcp proxy", func() {
			err := s.CreateResourceFromString(fmt.Sprintf(apisixRoute, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoute")

			s.RequestAssert(&scaffold.RequestAssert{
				Client: s.NewAPISIXClientWithTCPProxy(),
				Method: "GET",
				Path:   "/ip",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyContains("origin"),
				},
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Client: s.NewAPISIXClientWithTCPProxy(),
				Method: "GET",
				Path:   "/get",
				Headers: map[string]string{
					"x-my-header": "x-my-value",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyContains("x-my-value"),
				},
			})
		})
	})

	Context("UDP Proxy", func() {
		apisixRoute := `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-udp-route
spec:
  ingressClassName: %s
  stream:
  - name: rule1
    protocol: UDP
    match:
      ingressPort: 9200
    backend:
      serviceName: %s
      servicePort: %d
`
		It("stream udp proxy", func() {
			dnsSvc := s.NewCoreDNSService()
			err := s.CreateResourceFromString(fmt.Sprintf(apisixRoute, s.Namespace(), dnsSvc.Name, dnsSvc.Spec.Ports[0].Port))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoute")
			time.Sleep(20 * time.Second)

			svc := s.GetDataplaneService()

			// test dns query
			output, err := s.RunDigDNSClientFromK8s(fmt.Sprintf("@%s", svc.Name), "-p", "9200", "github.com")
			Expect(err).NotTo(HaveOccurred(), "dig github.com via apisix udp proxy")
			Expect(output).To(ContainSubstring("ADDITIONAL SECTION"))

			time.Sleep(3 * time.Second)
			output = s.GetDeploymentLogs(scaffold.CoreDNSDeployment)
			Expect(output).To(ContainSubstring("github.com. udp"))
		})
	})

	Context("Plugins", func() {
		It("MQTT", func() {
			//nolint:misspell // eclipse-mosquitto is the correct image name
			mqttDeploy := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mosquito
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mosquito
  template:
    metadata:
      labels:
        app: mosquito
    spec:
      containers:
      - name: mosquito
        image: eclipse-mosquitto:1.6
        livenessProbe:
          tcpSocket:
            port: 1883
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          tcpSocket:
            port: 1883
          initialDelaySeconds: 5
          periodSeconds: 10
        ports:
        - name: mosquito
          containerPort: 1883
          protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: mosquito
spec:
  selector:
    app: mosquito
  type: ClusterIP
  ports:
  - port: 1883
    targetPort: 1883
    protocol: TCP
`
			apisixRoute := `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: mqtt-route
spec:
  ingressClassName: %s
  stream:
  - name: rule1
    protocol: TCP
    match:
      ingressPort: 9100
    backend:
      serviceName: mosquito
      servicePort: 1883
    plugins:
    - name: mqtt-proxy
      enable: true
      config:
        protocol_name: MQTT
        protocol_level: 4
`
			err := s.CreateResourceFromString(mqttDeploy)
			Expect(err).NotTo(HaveOccurred(), "creating mosquito deployment")

			s.WaitUntilDeploymentAvailable("mosquito")

			err = s.CreateResourceFromString(fmt.Sprintf(apisixRoute, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoute")

			s.RetryAssertion(func() error {
				opts := mqtt.NewClientOptions()
				opts.AddBroker(fmt.Sprintf("tcp://%s", s.GetAPISIXTCPEndpoint()))
				mqttClient := mqtt.NewClient(opts)
				token := mqttClient.Connect()
				token.WaitTimeout(3 * time.Second)
				return token.Error()
			}).ShouldNot(HaveOccurred(), "connecting to mqtt proxy")
		})
	})
})
