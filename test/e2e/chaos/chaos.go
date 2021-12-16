package chaos

import (
	"fmt"
	"net/http"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("Chaos Testing", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta2",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.FContext("simulate apisix deployment restart", func() {
		ginkgo.Specify("ingress controller can synchronize rules normally after apisix recovery", func() {
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(0), "checking number of upstreams")
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			route1 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta2
kind: ApisixRoute
metadata:
  name: httpbin-route1
spec:
  http:
  - name: route1
    match:
      hosts:
      - httpbin.org
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route1))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of routes")
			s.RestartAPISIXDeploy()
			route2 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta2
kind: ApisixRoute
metadata:
  name: httpbin-route2
spec:
  http:
  - name: route2
    match:
      hosts:
      - httpbin.org
      paths:
      - /get
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route2))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "checking number of routes")
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
			s.NewAPISIXClient().GET("/get").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		})
	})

})
