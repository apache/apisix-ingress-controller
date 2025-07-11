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
			const total = 1000

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
      - /*
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
				text.WriteString(fmt.Sprintf(apisixRouteSpec, name, name))
				text.WriteString("\n---\n")
			}

			err := s.CreateResourceFromString(text.String())
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoutes")

			By("count time")
			now := time.Now()
			time.Sleep(30 * time.Second)

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
				if s.NewAPISIXClient().GET("/get").WithHeader("X-Route-Name", name).Expect().Raw().StatusCode == http.StatusOK {
					totalWorks++
					By(fmt.Sprintf("[%d/%d]%s works", totalWorks, total, name))
					continue
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
			// 		err := wait.PollUntilContextTimeout(context.Background(), 3*time.Second, 10*time.Minute, true, func(ctx context.Context) (done bool, err error) {
			// 			resp := s.NewAPISIXClient().GET("/get").WithHeader("X-Route-Name", name).Expect().Raw()
			// 			return resp.StatusCode == http.StatusOK, nil
			// 		})
			// 		Expect(err).NotTo(HaveOccurred())
			// 		By(fmt.Sprintf("ApisixRoute %s works", name))
			// 	}
			// 	go task(name)
			// }
			//
			// w.Wait()
			fmt.Printf("======2000 ApisixRoutes 生效时间为: %s =========", time.Since(now))
		})
	})
})

func getRouteName(i int) string {
	return fmt.Sprintf("test-route-%04d", i)
}
