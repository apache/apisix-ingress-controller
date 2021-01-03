package ingress

import (
	"github.com/api7/ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"time"
)

var _ = ginkgo.Describe("upstream expansion", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("create and then scale to 2 ", func() {
		apisixRoute := `
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  rules:
  - host: httpbin.com
    http:
      paths:
      - backend:
          serviceName: httpbin-service-e2e-test
          servicePort: 80
        path: /ip
`
		s.CreateApisixRouteByString(apisixRoute)

		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		s.ScaleHTTPBIN(2)
		time.Sleep(25 * time.Second)
		response, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "List upstreams error")
		assert.Equal(ginkgo.GinkgoT(), 2, len(response.Upstreams.Upstreams[0].UpstreamNodes.Nodes), "upstreams nodes not expect")
	})
})
