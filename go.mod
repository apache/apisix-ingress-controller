module github.com/apache/apisix-ingress-controller

go 1.13

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/hashicorp/go-memdb v1.0.4
	github.com/hashicorp/go-multierror v1.0.0
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/client_model v0.2.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20210415231046-e915ea6b2b7d
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/code-generator v0.21.1
	knative.dev/networking v0.0.0-20210512050647-ace2d3306f0b
	knative.dev/serving v0.23.0
)
