package ingress

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("Testing compare resources", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("Compare and find out the redundant objects in APISIX, and remove them", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta1
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
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))

		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		// scale Ingres Controller --replicas=0
		assert.Nil(ginkgo.GinkgoT(), s.ScaleIngressController(0), "scaling ingress controller instances = 0")
		// remove ApisixRoute resource
		assert.Nil(ginkgo.GinkgoT(), s.RemoveResourceByString(apisixRoute))
		// scale Ingres Controller --replicas=1
		assert.Nil(ginkgo.GinkgoT(), s.ScaleIngressController(1), "scaling ingress controller instances = 1")
		time.Sleep(15 * time.Second)
		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
		assert.Len(ginkgo.GinkgoT(), routes, 0, "route should be removed")
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups, 0, "upstream should be removed")
	})
})
