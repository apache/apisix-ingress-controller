module github.com/api7/ingress-controller/test/e2e

go 1.14

require (
	github.com/gavv/httpexpect/v2 v2.1.0
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/gruntwork-io/terratest v0.31.2
	github.com/gxthrj/apisix-ingress-types v0.1.3
	github.com/gxthrj/apisix-types v0.1.3
	github.com/gxthrj/seven v0.2.7
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
)

replace (
	github.com/gxthrj/apisix-ingress-types v0.1.3 => github.com/api7/ingress-types v0.1.3
	github.com/gxthrj/apisix-types v0.1.3 => github.com/api7/types v0.1.3
)
