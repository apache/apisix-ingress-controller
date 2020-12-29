module github.com/api7/ingress-controller

go 1.13

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/gxthrj/apisix-ingress-types v0.1.3
	github.com/gxthrj/apisix-types v0.1.3
	github.com/hashicorp/go-memdb v1.0.4
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.8.1 // indirect
	github.com/prometheus/client_golang v0.9.3
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.4.0
	github.com/yudai/gojsondiff v1.0.0
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yudai/pp v2.0.1+incompatible // indirect
	go.uber.org/zap v1.13.0
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.0.0-20190819141258-3544db3b9e44
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190819141724-e14f31a72a77
)

replace github.com/gxthrj/apisix-ingress-types v0.1.3 => github.com/api7/ingress-types v0.1.3

replace github.com/gxthrj/apisix-types v0.1.3 => github.com/api7/types v0.1.3
