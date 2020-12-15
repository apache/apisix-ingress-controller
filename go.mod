module github.com/api7/ingress-controller

go 1.13

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gxthrj/apisix-ingress-types v0.1.3
	github.com/gxthrj/apisix-types v0.1.3
	github.com/gxthrj/seven v0.2.7
	github.com/onsi/ginkgo v1.11.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.4.0
	go.uber.org/zap v1.13.0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.0.0-20190819141258-3544db3b9e44
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190819141724-e14f31a72a77
)

replace github.com/gxthrj/apisix-ingress-types v0.1.3 => github.com/api7/ingress-types v0.1.3

replace github.com/gxthrj/apisix-types v0.1.3 => github.com/api7/types v0.1.3
