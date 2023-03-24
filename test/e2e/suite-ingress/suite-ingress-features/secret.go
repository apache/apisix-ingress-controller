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
	"io/ioutil"
	"log"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress-features: secret controller", func() {
	apisixTlsSuites := func(s *scaffold.Scaffold) {
		ginkgo.It("should update SSL if secret referenced by ApisixTls is created later", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - api6.com
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

			secretName := "test-apisix-tls"
			// create ApisixTls resource
			tlsName := "tls-name"
			host := "api6.com"
			err := s.NewApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
			time.Sleep(10 * time.Second)

			// create secret later than ApisixTls
			certBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/cert.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching certificate error")
				log.Fatal(err)
			}
			cert := string(certBytes)

			keyBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/key.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching key error")
				log.Fatal(err)
			}
			key := string(keyBytes)
			// key compare
			keyCompare := "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"
			// create secret
			err = s.NewSecret(secretName, cert, key)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)

			// verify SSL resource
			tls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
			assert.Equal(ginkgo.GinkgoT(), keyCompare, tls[0].Key, "tls key not expect")

			// check DP
			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()
		})

		ginkgo.It("should update SSL if secret referenced by ApisixTls is updated", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - api6.com
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

			secretName := "test-apisix-tls"
			certBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/cert1.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching certificate error")
				log.Fatal(err)
			}
			cert := string(certBytes)

			keyBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/key1.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching key error")
				log.Fatal(err)
			}
			key := string(keyBytes)
			// key compare
			keyCompare := "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"
			// create secret
			err = s.NewSecret(secretName, cert, key)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// create ApisixTls resource
			tlsName := "tls-name"
			host := "api6.com"
			err = s.NewApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
			assert.Equal(ginkgo.GinkgoT(), keyCompare, tls[0].Key, "tls key not expect")

			// check DP
			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

			certUpdateBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/certUpdate1.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching updated certificate error")
				log.Fatal(err)
			}
			certUpdate := string(certUpdateBytes)

			keyUpdateBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/keyUpdate1.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching updated key error")
				log.Fatal(err)
			}
			keyUpdate := string(keyUpdateBytes)

			keyCompareUpdate := "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"
			// key update compare
			err = s.NewSecret(secretName, certUpdate, keyUpdate)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tlsUpdate, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
			assert.Len(ginkgo.GinkgoT(), tlsUpdate, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), certUpdate, tlsUpdate[0].Cert, "tls cert not expect")
			assert.Equal(ginkgo.GinkgoT(), keyCompareUpdate, tlsUpdate[0].Key, "tls key not expect")
			// check DP
			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

			// delete ApisixTls
			err = s.DeleteApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "delete tls error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err = s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 0, "tls number not expect")
		})

		ginkgo.It("should be able to handle a kube style SSL secret", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - api6.com
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))

			secretName := "test-apisix-tls"
			certBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/cert2.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching certificate error")
				log.Fatal(err)
			}
			cert := string(certBytes)

			keyBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/key2.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching key error")
				log.Fatal(err)
			}
			key := string(keyBytes)
			// key compare
			keyCompare := "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"
			// create kube secret
			err = s.NewKubeTlsSecret(secretName, cert, key)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// create ApisixTls resource
			tlsName := "tls-name"
			host := "api6.com"
			err = s.NewApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
			assert.Equal(ginkgo.GinkgoT(), keyCompare, tls[0].Key, "tls key not expect")

			// check DP
			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

			certUpdateBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/certUpdate2.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching updated certificate error")
				log.Fatal(err)
			}
			certUpdate := string(certUpdateBytes)

			keyUpdateBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/keyUpdate2.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching updated key error")
				log.Fatal(err)
			}
			keyUpdate := string(keyUpdateBytes)

			keyCompareUpdate := "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"
			// key update compare
			err = s.NewKubeTlsSecret(secretName, certUpdate, keyUpdate)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tlsUpdate, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
			assert.Len(ginkgo.GinkgoT(), tlsUpdate, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), certUpdate, tlsUpdate[0].Cert, "tls cert not expect")
			assert.Equal(ginkgo.GinkgoT(), keyCompareUpdate, tlsUpdate[0].Key, "tls key not expect")
			// check DP
			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

			// delete ApisixTls
			err = s.DeleteApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "delete tls error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err = s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 0, "tls number not expect")
		})
	}

	ingressSuites := func(s *scaffold.Scaffold) {
		ginkgo.It("should update SSL if secret referenced by Ingress is updated", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			ingress := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
  name: ingress-route
spec:
  rules:
  - host: api6.com
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ingress))

			secretName := "test-apisix-tls"
			certBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/cert3.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching certificate error")
				log.Fatal(err)
			}
			cert := string(certBytes)

			keyBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/key3.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching key error")
				log.Fatal(err)
			}
			key := string(keyBytes)
			// key compare
			keyCompare := "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"
			// create secret
			err = s.NewSecret(secretName, cert, key)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// create ApisixTls resource
			tlsName := "tls-name"
			host := "api6.com"
			err = s.NewApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
			assert.Equal(ginkgo.GinkgoT(), keyCompare, tls[0].Key, "tls key not expect")

			// check DP
			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

			certUpdateBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/certUpdate3.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching updated certificate error")
				log.Fatal(err)
			}
			certUpdate := string(certUpdateBytes)

			keyUpdateBytes, err := ioutil.ReadFile("test/e2e/testbackend/tls/keyUpdate3.pem")
			if err != nil {
				assert.Nil(ginkgo.GinkgoT(), err, "fetching updated key error")
				log.Fatal(err)
			}
			keyUpdate := string(keyUpdateBytes)

			keyCompareUpdate := "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"
			// key update compare
			err = s.NewSecret(secretName, certUpdate, keyUpdate)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tlsUpdate, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
			assert.Len(ginkgo.GinkgoT(), tlsUpdate, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), certUpdate, tlsUpdate[0].Cert, "tls cert not expect")
			assert.Equal(ginkgo.GinkgoT(), keyCompareUpdate, tlsUpdate[0].Key, "tls key not expect")
			// check DP
			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK).Body().Raw()

			// delete ApisixTls
			err = s.DeleteApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "delete tls error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err = s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 0, "tls number not expect")
		})
	}

	ginkgo.Describe("suite-ingress-features: scaffold v2beta3", func() {
		apisixTlsSuites(scaffold.NewDefaultV2beta3Scaffold())
	})
	ginkgo.Describe("suite-ingress-features: scaffold v2", func() {
		s := scaffold.NewDefaultV2Scaffold()
		apisixTlsSuites(s)
		ingressSuites(s)
	})
})
