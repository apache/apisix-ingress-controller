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

package v1alpha1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GatewayProxy control plane TLS", Label("apisix.apache.org", "v1alpha1", "gatewayproxy"), func() {
	var s = scaffold.NewDefaultScaffold()

	// The admin API only listens on plain HTTP, so an openresty in front of it
	// stands in for a control plane published over TLS. Its certificate is
	// signed by a CA generated per test, which is exactly the case caBundle
	// exists for: nothing in the system trust store can verify it.
	const adminTLSProxySpec = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: admin-tls
data:
  nginx.conf: |
    worker_processes 1;
    pid /run/nginx.pid;
    events {
      worker_connections 1024;
    }
    http {
      # the standalone provider PUTs the whole configuration through here
      client_max_body_size 32m;
      server {
        listen 9543 ssl;
        ssl_certificate /etc/nginx/ssl/tls.crt;
        ssl_certificate_key /etc/nginx/ssl/tls.key;
        location / {
          proxy_pass %s;
          proxy_http_version 1.1;
        }
      }
      server {
        listen 9544;
        location /healthz {
          return 200 'ok';
        }
      }
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: admin-tls
spec:
  replicas: 1
  selector:
    matchLabels:
      app: admin-tls
  template:
    metadata:
      labels:
        app: admin-tls
    spec:
      volumes:
        - name: config
          configMap:
            name: admin-tls
        - name: ssl
          secret:
            secretName: admin-tls
      containers:
        - name: admin-tls
          image: "openresty/openresty:1.27.1.2-4-bullseye-fat"
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 9543
              name: https
              protocol: TCP
          readinessProbe:
            httpGet:
              path: /healthz
              port: 9544
            initialDelaySeconds: 2
            periodSeconds: 2
          volumeMounts:
            - mountPath: /usr/local/openresty/nginx/conf/nginx.conf
              name: config
              subPath: nginx.conf
            - mountPath: /etc/nginx/ssl
              name: ssl
---
apiVersion: v1
kind: Service
metadata:
  name: admin-tls
spec:
  selector:
    app: admin-tls
  ports:
    - name: https
      port: 9543
      protocol: TCP
      targetPort: 9543
`

	const gatewayProxySpec = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config-tls
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      tlsVerify: true
%s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

	const gatewayClassSpec = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

	const gatewaySpec = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config-tls
`

	const httpRouteSpec = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin-tls-cp
spec:
  parentRefs:
  - name: %s
  hostnames:
  - "%s"
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

	// indent a PEM block so it survives being inlined into the YAML above
	indent := func(pem string) string {
		lines := strings.Split(strings.TrimRight(pem, "\n"), "\n")
		for i, line := range lines {
			lines[i] = "        " + line
		}
		return strings.Join(lines, "\n")
	}

	var caCert string

	BeforeEach(func() {
		By("generate a CA and a server certificate for the TLS control plane")
		// the ADC sidecar dials the proxy by its in-cluster name, so that is
		// what the certificate has to be valid for
		var serverCert, serverKey string
		caCert, serverCert, serverKey = generateSignedCert([]string{
			"admin-tls",
			fmt.Sprintf("admin-tls.%s", s.Namespace()),
			fmt.Sprintf("admin-tls.%s.svc", s.Namespace()),
		})

		err := s.NewKubeTlsSecret("admin-tls", serverCert, serverKey)
		Expect(err).NotTo(HaveOccurred(), "creating the server certificate secret")

		By("deploy a TLS terminator in front of the admin API")
		err = s.CreateResourceFromString(fmt.Sprintf(adminTLSProxySpec, s.Deployer.GetAdminEndpoint()))
		Expect(err).NotTo(HaveOccurred(), "creating the TLS control plane")
		Expect(framework.WaitPodsAvailable(s.GinkgoT, s.KubeOpts(), metav1.ListOptions{
			LabelSelector: "app=admin-tls",
		})).NotTo(HaveOccurred(), "waiting for the TLS control plane")
	})

	// routes the gateway through a GatewayProxy that reaches the control plane
	// over TLS, with caBundle set to caBundleField
	attachRoute := func(hostname, caBundleField string) {
		By("create GatewayProxy")
		endpoint := fmt.Sprintf("https://admin-tls.%s:9543", s.Namespace())
		err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxySpec, endpoint, caBundleField, s.AdminKey()))
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create GatewayClass")
		gatewayClassName := fmt.Sprintf("%s-tls", s.Namespace())
		err = s.CreateResourceFromString(fmt.Sprintf(gatewayClassSpec, gatewayClassName, s.GetControllerName()))
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")

		By("create Gateway")
		gatewayName := fmt.Sprintf("%s-tls", s.Namespace())
		err = s.CreateResourceFromString(fmt.Sprintf(gatewaySpec, gatewayName, gatewayClassName))
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")

		By("create HTTPRoute")
		err = s.CreateResourceFromString(fmt.Sprintf(httpRouteSpec, gatewayName, hostname))
		Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute")
	}

	It("syncs to a private-CA control plane when caBundle is trusted", func() {
		attachRoute("httpbin-tls-cp.org", "      caBundle: |\n"+indent(caCert))

		By("the route is programmed, so the sync got through TLS verification")
		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET",
			Path:   "/get",
			Host:   "httpbin-tls-cp.org",
			Check:  scaffold.WithExpectedStatus(http.StatusOK),
		})
	})

	It("fails to sync when caBundle is missing", func() {
		attachRoute("httpbin-tls-cp-untrusted.org", "")

		By("the control plane certificate cannot be verified")
		// look back as well as forward: the sync can fail before the stream opens
		s.WaitControllerManagerLog("unable to verify the first certificate", 60, time.Minute)

		By("so the route is never programmed")
		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET",
			Path:   "/get",
			Host:   "httpbin-tls-cp-untrusted.org",
			Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
		})
	})

	It("rejects a caBundle that is not a certificate", func() {
		By("create GatewayProxy with an unparseable caBundle")
		endpoint := fmt.Sprintf("https://admin-tls.%s:9543", s.Namespace())
		output, err := s.CreateResourceFromStringAndGetOutput(
			fmt.Sprintf(gatewayProxySpec, endpoint, "      caBundle: not-a-certificate", s.AdminKey()),
		)
		Expect(err).To(HaveOccurred(), "the API server should reject it")
		Expect(output + err.Error()).To(ContainSubstring("caBundle must be a PEM-encoded certificate"))
	})
})

// generateSignedCert returns a CA and a server certificate signed by it. The
// scaffold's GenerateMACert gives both the same subject, which OpenSSL reads as
// a self-issued certificate rather than one chaining to the CA, so a strict TLS
// client rejects it before it ever looks at the bundle.
func generateSignedCert(dnsNames []string) (caCertPEM, serverCertPEM, serverKeyPEM string) {
	serial := func() *big.Int {
		n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
		Expect(err).NotTo(HaveOccurred())
		return n
	}
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())
	caTmpl := &x509.Certificate{
		SerialNumber:          serial(),
		Subject:               pkix.Name{CommonName: "apisix-ingress-e2e-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	Expect(err).NotTo(HaveOccurred())

	serverKeyECDSA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())
	serverTmpl := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: dnsNames[0]},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dnsNames,
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTmpl, caTmpl, &serverKeyECDSA.PublicKey, caKey)
	Expect(err).NotTo(HaveOccurred())

	keyDER, err := x509.MarshalPKCS8PrivateKey(serverKeyECDSA)
	Expect(err).NotTo(HaveOccurred())

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})),
		string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
}
