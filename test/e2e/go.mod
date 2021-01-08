module github.com/api7/ingress-controller/test/e2e

go 1.14

require (
	github.com/api7/ingress-controller v0.0.0-20210105024109-72e53386de5a
	github.com/gavv/httpexpect/v2 v2.1.0
	github.com/gruntwork-io/terratest v0.31.2
	github.com/onsi/ginkgo v1.14.2
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
)

replace github.com/gxthrj/apisix-ingress-types v0.1.3 => github.com/api7/ingress-types v0.1.3

replace github.com/api7/ingress-controller => ../../
