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

package plugins

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress: mqtt-proxy plugin", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2",
	}
	s := scaffold.NewScaffold(opts)
	// setup mosquito service
	ginkgo.It("stream mqtt proxy", func() {
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(`
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
`))
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(`
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
`))
		s.EnsureNumEndpointsReady(ginkgo.GinkgoT(), "mosquito", 1)
		// setup Apisix Route for mqtt proxy
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: mqtt-route
spec:
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
`)

		time.Sleep(12 * time.Second)
		err := s.CreateResourceFromString(apisixRoute)
		assert.Nil(ginkgo.GinkgoT(), err)
		defer func() {
			err := s.RemoveResourceByString(apisixRoute)
			assert.Nil(ginkgo.GinkgoT(), err)
		}()

		err = s.EnsureNumApisixStreamRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		sr, err := s.ListApisixStreamRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), sr, 1)
		assert.Equal(ginkgo.GinkgoT(), sr[0].ServerPort, int32(9100))
		// test mqtt protocol
		c := s.NewMQTTClient()
		token := c.Connect()
		token.WaitTimeout(3 * time.Second)
		assert.Nil(ginkgo.GinkgoT(), token.Error(), "mqtt connection error")
	})
})
