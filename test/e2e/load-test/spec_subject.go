package load

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

const gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      service:
        name: %s
        port: 9180
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

const ingressClassYaml = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: %s
    scope: "Namespace"
`

var _ = Describe("Load Test", func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
	)

	BeforeEach(func() {
		By("config ingress controller ProviderSyncPeriod")
		s.DeployIngress(framework.IngressDeployOpts{
			ControllerName:     s.GetControllerName(),
			ProviderType:       framework.ProviderType,
			ProviderSyncPeriod: 1 * time.Second,
			Namespace:          s.Namespace(),
			Replicas:           1,
		})

		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.ProviderType, s.AdminKey())
		err := s.CreateResourceFromStringWithNamespace(gatewayProxy, s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(ingressClassYaml, s.Namespace()), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	Context("Load Test 2000 ApisixRoute", func() {
		It("test 2000 ApisixRoute", func() {
			const total = 2000

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /get
      exprs:
      - subject:
          scope: Header
          name: X-Route-Name
        op: Equal
        value: %s
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`

			By(fmt.Sprintf("prepare %d ApisixRoutes", total))
			var text = bytes.NewBuffer(nil)
			for i := range total {
				name := getRouteName(i)
				_, err := fmt.Fprintf(text, apisixRouteSpec, name, name)
				Expect(err).NotTo(HaveOccurred())
				text.WriteString("\n---\n")
			}

			err := s.CreateResourceFromString(text.String())
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoutes")

			By("count time")
			now := time.Now()

			var c = make(chan int, total)
			for i := range total {
				c <- i
			}

			var totalWorks = 0
			for {
				if len(c) == 0 {
					close(c)
					break
				}
				i := <-c
				name := getRouteName(i)
				By(fmt.Sprintf("[%d/%d]try to verify %s", totalWorks, total, name))
				statusCode := s.NewAPISIXClient().GET("/get").WithHeader("X-Route-Name", name).Expect().Raw().StatusCode
				if statusCode == http.StatusOK {
					totalWorks++
					By(fmt.Sprintf("[%d/%d]%s works", totalWorks, total, name))
					continue
				}
				if statusCode != http.StatusNotFound {
					Fail("status code should be 404")
				}
				time.Sleep(100 * time.Millisecond)
				c <- i
			}

			// w := sync.WaitGroup{}
			// for i := range total {
			// 	time.Sleep(100 * time.Millisecond)
			// 	name := getRouteName(i)
			// 	w.Add(1)
			// 	task := func(name string) {
			// 		defer w.Done()
			// 		By(fmt.Sprintf("to check ApisixRoute %s works", name))
			// 		err := wait.PollUntilContextTimeout(context.Background(), 100*time.Millisecond, time.Minute, true, func(ctx context.Context) (done bool, err error) {
			// 			resp := s.NewAPISIXClient().GET("/get").WithHeader("X-Route-Name", name).Expect().Raw()
			// 			return resp.StatusCode == http.StatusOK, nil
			// 		})
			// 		Expect(err).NotTo(HaveOccurred())
			// 		By(fmt.Sprintf("ApisixRoute %s works", name))
			// 	}
			// 	go task(name)
			// }
			// w.Wait()

			fmt.Printf("======%d ApisixRoutes takes effect for: %s =========\n", total, time.Since(now))

			By("Test the time required for an ApisixRoute update to take effect")
			var apisixRouteSpec0 = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /headers
      exprs:
      - subject:
          scope: Header
          name: X-Route-Name
        op: Equal
        value: %s
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			name := getRouteName(10)
			err = s.CreateResourceFromString(fmt.Sprintf(apisixRouteSpec0, name, name))
			Expect(err).NotTo(HaveOccurred())
			now = time.Now()
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/headers").WithHeader("X-Route-Name", name).Expect().Raw().StatusCode
			}).WithTimeout(time.Minute).ProbeEvery(100 * time.Millisecond).Should(Equal(http.StatusOK))
			fmt.Printf("====== 更新 ApisixRoute 生效时间为: %s =========\n", time.Since(now))
		})
	})
})

func getRouteName(i int) string {
	return fmt.Sprintf("test-route-%04d", i)
}
