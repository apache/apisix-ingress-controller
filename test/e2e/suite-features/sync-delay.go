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

package features

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

func getApisixResourceRequestsCount(s *scaffold.Scaffold, res string) int {
	pods, err := s.GetIngressPodDetails()
	assert.Nil(ginkgo.GinkgoT(), err, "get ingress pod")
	assert.True(ginkgo.GinkgoT(), len(pods) >= 1, "get ingress pod")

	output, err := s.Exec(pods[0].Name, "ingress-apisix-controller-deployment-e2e-test",
		fmt.Sprintf("wget -q -O - localhost:8080/metrics | grep apisix_ingress_controller_apisix_requests | grep 'op=\"write\"' | grep 'resource=\"%v\"'", res),
	)
	if err != nil {
		log.Errorf("failed to get metrics: %v, %v; output: %v", err.Error(), string(err.(*exec.ExitError).Stderr), output)
	} else {
		log.Infof("output: %v", output)
	}
	assert.Nil(ginkgo.GinkgoT(), err, "get metrics from controller")

	// make sure the output is grep-ed
	assert.False(ginkgo.GinkgoT(), strings.Contains(output, "promhttp_metric_handler_requests_total"))
	assert.True(ginkgo.GinkgoT(), strings.Contains(output, "apisix_ingress_controller_apisix_requests"))
	assert.True(ginkgo.GinkgoT(), strings.Contains(output, fmt.Sprintf("resource=\"%v\"", res)))

	arr := strings.Split(output, " ")
	if len(arr) == 0 {
		ginkgo.Fail("unexpected metrics output: "+output, 1)
		return -1
	}
	i, err := strconv.ParseInt(arr[len(arr)-1], 10, 64)
	assert.Nil(ginkgo.GinkgoT(), err, "parse metrics")
	return int(i)
}

var _ = ginkgo.Describe("suite-features: sync delay", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("check resource request count", func() {
			if s.IsEtcdServer() {
				ginkgo.Skip("Does not support etcdserver mode, temporarily skipping test cases, waiting for fix")
			}
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "create ApisixRoute")
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of upstreams")

			arStream := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-tcp-route
spec:
  stream:
  - name: rule1
    protocol: TCP
    match:
      ingressPort: 9100
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(arStream), "create ApisixRoute (stream)")

			hostA := "a.test.com"
			secretA := "server-secret-a"
			serverCertA, serverKeyA := s.GenerateCert(ginkgo.GinkgoT(), []string{hostA})
			err = s.NewSecret(secretA, serverCertA.String(), serverKeyA.String())
			assert.Nil(ginkgo.GinkgoT(), err, "create server cert secret 'a' error")
			err = s.NewApisixTls("tls-server-a", hostA, secretA)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixTls 'a' error")

			ac := `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: foo
spec:
  authParameter:
    jwtAuth:
      value:
        key: foo-key
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "create ApisixConsumer")

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
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "create ApisixGlobalRule")

			defer func() {
				s.DeleteResource("ApisixGlobalRule", "test-agr-1")
			}()

			apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: test-apc-1
spec:
 plugins:
 - name: cors
   enable: true
`)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc), "create ApisixPluginConfig")

			// ensure resources exist, so ResourceSync will be triggered
			time.Sleep(6 * time.Second)

			routes, err := s.ListApisixRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), len(routes) > 0)

			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), len(ups) > 0)

			srs, err := s.ListApisixStreamRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), len(srs) > 0)

			ssls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), len(ssls) > 0)

			consumers, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), len(consumers) > 0)

			grs, err := s.ListApisixGlobalRules()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), len(grs) > 0)

			pcs, err := s.ListApisixPluginConfig()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.True(ginkgo.GinkgoT(), len(pcs) > 0)

			// Check request counts
			resTypes := []string{"route", "upstream", "ssl", "streamRoute",
				"consumer", "globalRule", "pluginConfig",
			}

			countersBeforeWait := map[string]int{}
			for _, resType := range resTypes {
				countersBeforeWait[resType] = getApisixResourceRequestsCount(s, resType)
			}

			sleepMins := 2
			log.Infof("before sleep requests count: %v, wait for %v mins ...", countersBeforeWait, sleepMins)
			time.Sleep(time.Second * time.Duration(60*sleepMins+10))

			countersExpectedIncrement := map[string]int{
				"route":       sleepMins,
				"upstream":    sleepMins * 2,
				"streamRoute": sleepMins,

				// these resources have only 1 instance, so the last triggered sync event
				// haven't been processed yet
				"ssl":          sleepMins,
				"consumer":     sleepMins,
				"globalRule":   sleepMins,
				"pluginConfig": sleepMins,
			}

			countersAfterWait := map[string]int{}
			for _, resType := range resTypes {
				countersAfterWait[resType] = getApisixResourceRequestsCount(s, resType)
				expected := countersBeforeWait[resType] + countersExpectedIncrement[resType]
				if countersAfterWait[resType] < expected {
					log.Errorf("request count: %v expect %v but got %v", resType, expected, countersAfterWait[resType])
				}
			}

			log.Infof("after sleep requests count: %v, wait for %v mins", countersAfterWait, sleepMins)

			for _, resType := range resTypes {
				expected := countersBeforeWait[resType] + countersExpectedIncrement[resType]
				assert.NotEqual(ginkgo.GinkgoT(), countersBeforeWait[resType], countersAfterWait[resType], resType+" request count")
				assert.GreaterOrEqual(ginkgo.GinkgoT(), countersAfterWait[resType], expected, resType+" request count")
			}
		})

		ginkgo.It("check multiple resources request count", func() {
			if s.IsEtcdServer() {
				ginkgo.Skip("Does not support etcdserver mode, temporarily skipping test cases, waiting for fix")
			}
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
---
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-2
spec:
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world2!!"
---
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-agr-3
spec:
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello, world3!!"
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(agr), "create ApisixGlobalRule")

			defer func() {
				s.DeleteResource("ApisixGlobalRule", "test-agr-1")
				s.DeleteResource("ApisixGlobalRule", "test-agr-2")
				s.DeleteResource("ApisixGlobalRule", "test-agr-3")
			}()

			// ensure resources exist, so ResourceSync will be triggered
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixGlobalRules()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), 3, len(grs))

			// Check request counts
			resTypes := []string{"globalRule"}

			countersBeforeWait := map[string]int{}
			for _, resType := range resTypes {
				countersBeforeWait[resType] = getApisixResourceRequestsCount(s, resType)
			}

			sleepMins := 2
			log.Infof("before sleep requests count: %v, wait for %v mins ...", countersBeforeWait, sleepMins)
			time.Sleep(time.Second * time.Duration(60*sleepMins+3))

			countersExpectedIncrement := map[string]int{
				"globalRule": (sleepMins)*3 - 2,
			}

			countersAfterWait := map[string]int{}
			for _, resType := range resTypes {
				countersAfterWait[resType] = getApisixResourceRequestsCount(s, resType)
				expected := countersBeforeWait[resType] + countersExpectedIncrement[resType]
				if countersAfterWait[resType] <= expected {
					log.Errorf("request count: %v expect %v but got %v", resType, expected, countersAfterWait[resType])
				}
			}
			log.Infof("after sleep requests count: %v, wait for %v mins", countersAfterWait, sleepMins)

			for _, resType := range resTypes {
				expected := countersBeforeWait[resType] + countersExpectedIncrement[resType]
				assert.NotEqual(ginkgo.GinkgoT(), countersBeforeWait[resType], countersAfterWait[resType], resType+" request count")
				assert.GreaterOrEqual(ginkgo.GinkgoT(), countersAfterWait[resType], expected, resType+" request count")
			}
		})
	}

	ginkgo.Describe("scaffold v2", func() {
		suites(scaffold.NewV2Scaffold(&scaffold.Options{
			ApisixResourceSyncInterval:   "60s",
			ApisixResourceSyncComparison: "false",
		}))
	})
})
