module github.com/apache/apisix-ingress-controller/test/e2e

go 1.16

require (
	github.com/apache/apisix-ingress-controller v0.0.0-20210105024109-72e53386de5a
	github.com/apache/apisix-ingress-controller/test/e2e/testbackend v0.0.0
	github.com/gavv/httpexpect/v2 v2.2.0
	github.com/gorilla/websocket v1.4.2
	github.com/gruntwork-io/terratest v0.32.8
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/client-go v0.22.4
)

replace github.com/apache/apisix-ingress-controller => ../../

replace github.com/apache/apisix-ingress-controller/test/e2e/testbackend => ./testbackend
