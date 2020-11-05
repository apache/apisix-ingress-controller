module github.com/api7/ingress-controller

go 1.13

require (
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20180511133405-39ca1b05acc7 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/uuid v1.1.1 // indirect
	github.com/gxthrj/apisix-ingress-types v0.1.2
	github.com/gxthrj/apisix-types v0.1.0
	github.com/gxthrj/seven v0.1.9
	github.com/julienschmidt/httprouter v1.3.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	github.com/tidwall/gjson v1.3.5
	go.uber.org/zap v1.13.0 // indirect
	google.golang.org/genproto v0.0.0-20191205163323-51378566eb59 // indirect
	google.golang.org/grpc v1.25.1 // indirect
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20190819141258-3544db3b9e44
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190819141724-e14f31a72a77
)

replace github.com/gxthrj/apisix-ingress-types v0.1.2 => github.com/api7/ingress-types v0.1.2

replace github.com/gxthrj/apisix-types v0.1.0 => github.com/api7/types v0.1.0
