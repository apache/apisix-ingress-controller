package features

import (
	"fmt"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"net/http"
	"time"
)

var _ = ginkgo.FDescribe("suite-gatewayapi: HTTP Route", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("Basic HTTPRoute with 1 Rule 1 Match 1 BackendRef", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		time.Sleep(time.Second * 10)
		_, _ = backendSvc, backendPorts
		route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: HTTPRoute
metadata:
  name: basic-http-route
spec:
  hostnames: ["httpbin.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /ip
    backendRefs:
    - name: httpbin
      port: 80
`)

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)
	})
})
