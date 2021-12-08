module github.com/apache/apisix-ingress-controller

go 1.16

require (
	github.com/fsnotify/fsnotify v1.5.0 // indirect
	github.com/gin-gonic/gin v1.6.3
	github.com/google/uuid v1.2.0 // indirect
	github.com/hashicorp/go-memdb v1.0.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/slok/kubewebhook/v2 v2.1.0
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/xeipuuv/gojsonschema v1.2.0
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.18.1
	golang.org/x/net v0.0.0-20210520170846-37e1c6afe023
	golang.org/x/sys v0.0.0-20210819072135-bce67f096156 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/code-generator v0.22.0
	sigs.k8s.io/gateway-api v0.4.0
)
