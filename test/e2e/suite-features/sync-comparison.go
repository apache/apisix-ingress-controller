package features

import (
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-features: sync comparison", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.FIt("check Route resource request count", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
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
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of upstreams")

			// TODO When ingress controller can feedback the lifecycle of CRDs to the
			// status field, we can poll it rather than sleeping.
			time.Sleep(3 * time.Second)

			resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("origin")

			time.Sleep(time.Hour)
		})
	}

	//ginkgo.Describe("suite-features: scaffold v2beta3", func() {
	//	suites(scaffold.NewV2beta3Scaffold(&scaffold.Options{
	//		ApisixResourceSyncInterval: "60s",
	//	}))
	//})
	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewV2Scaffold(&scaffold.Options{
			ApisixResourceSyncInterval: "60s",
		}))
	})
})
