package annotations

import (
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-annotations: allow-http-methods annotations", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("enable in ingress networking/v1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/allow-http-methods: POST,PUT
  name: ingress-v1
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

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusMethodNotAllowed)
	})

	ginkgo.It("enable in ingress networking/v1beta1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/allow-http-methods: POST,PUT
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusMethodNotAllowed)
	})

	ginkgo.It("enable in ingress extensions/v1beta1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/allow-http-methods: POST,PUT
  name: ingress-extensions-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusMethodNotAllowed)
	})
})

var _ = ginkgo.Describe("suite-annotations: blocklist-http-methods annotations", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("enable in ingress networking/v1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/allow-http-methods: GET
  name: ingress-v1
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

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusMethodNotAllowed)
	})

	ginkgo.It("enable in ingress networking/v1beta1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/allow-http-methods: GET
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusMethodNotAllowed)
	})

	ginkgo.It("enable in ingress extensions/v1beta1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/allow-http-methods: GET
  name: ingress-extensions-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusMethodNotAllowed)
	})
})
