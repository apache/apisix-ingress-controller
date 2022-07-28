package translation

import (
	listerscorev1 "k8s.io/client-go/listers/core/v1"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type translator struct {
	KubeProvider  k8s.Provider
	PodLister     listerscorev1.PodLister
	ServiceLister listerscorev1.ServiceLister
	SecretLister  listerscorev1.SecretLister

	translation.Translator
}

type ApisixTranslator interface {
	// TranslateUpstreamConfigV2beta3 translates ApisixUpstreamConfig (part of ApisixUpstream)
	// to APISIX Upstream, it doesn't fill the the Upstream metadata and nodes.
	TranslateUpstreamConfigV2beta3(*configv2beta3.ApisixUpstreamConfig) (*apisixv1.Upstream, error)
	// TranslateUpstreamConfigV2 translates ApisixUpstreamConfig (part of ApisixUpstream)
	// to APISIX Upstream, it doesn't fill the the Upstream metadata and nodes.
	TranslateUpstreamConfigV2(*configv2.ApisixUpstreamConfig) (*apisixv1.Upstream, error)
	// TranslateRouteV2beta2 translates the configv2beta2.ApisixRoute object into several Route,
	// and Upstream resources.
	TranslateRouteV2beta2(*configv2beta2.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2beta2NotStrictly translates the configv2beta2.ApisixRoute object into several Route,
	// and Upstream  resources not strictly, only used for delete event.
	TranslateRouteV2beta2NotStrictly(*configv2beta2.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2beta3 translates the configv2beta3.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources.
	TranslateRouteV2beta3(*configv2beta3.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2beta3NotStrictly translates the configv2beta3.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources not strictly, only used for delete event.
	TranslateRouteV2beta3NotStrictly(*configv2beta3.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2 translates the configv2.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources.
	TranslateRouteV2(*configv2.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateRouteV2NotStrictly translates the configv2.ApisixRoute object into several Route,
	// Upstream and PluginConfig resources not strictly, only used for delete event.
	TranslateRouteV2NotStrictly(*configv2.ApisixRoute) (*translation.TranslateContext, error)
	// TranslateSSLV2Beta3 translates the configv2beta3.ApisixTls object into the APISIX SSL resource.
	TranslateSSLV2Beta3(*configv2beta3.ApisixTls) (*apisixv1.Ssl, error)
	// TranslateSSLV2 translates the configv2.ApisixTls object into the APISIX SSL resource.
	TranslateSSLV2(*configv2.ApisixTls) (*apisixv1.Ssl, error)
	// TranslateClusterConfig translates the configv2beta3.ApisixClusterConfig object into the APISIX
	// Global Rule resource.
	TranslateClusterConfigV2beta3(*configv2beta3.ApisixClusterConfig) (*apisixv1.GlobalRule, error)
	// TranslateClusterConfigV2 translates the configv2.ApisixClusterConfig object into the APISIX
	// Global Rule resource.
	TranslateClusterConfigV2(*configv2.ApisixClusterConfig) (*apisixv1.GlobalRule, error)
	// TranslateApisixConsumer translates the configv2beta3.APisixConsumer object into the APISIX Consumer
	// resource.
	TranslateApisixConsumerV2beta3(*configv2beta3.ApisixConsumer) (*apisixv1.Consumer, error)
	// TranslateApisixConsumerV2 translates the configv2beta3.APisixConsumer object into the APISIX Consumer
	// resource.
	TranslateApisixConsumerV2(ac *configv2.ApisixConsumer) (*apisixv1.Consumer, error)
	// TranslatePluginConfigV2beta3 translates the configv2.ApisixPluginConfig object into several PluginConfig
	// resources.
	TranslatePluginConfigV2beta3(*configv2beta3.ApisixPluginConfig) (*translation.TranslateContext, error)
	// TranslatePluginConfigV2beta3NotStrictly translates the configv2beta3.ApisixPluginConfig object into several PluginConfig
	// resources not strictly, only used for delete event.
	TranslatePluginConfigV2beta3NotStrictly(*configv2beta3.ApisixPluginConfig) (*translation.TranslateContext, error)
	// TranslatePluginConfigV2 translates the configv2.ApisixPluginConfig object into several PluginConfig
	// resources.
	TranslatePluginConfigV2(*configv2.ApisixPluginConfig) (*translation.TranslateContext, error)
	// TranslatePluginConfigV2NotStrictly translates the configv2.ApisixPluginConfig object into several PluginConfig
	// resources not strictly, only used for delete event.
	TranslatePluginConfigV2NotStrictly(*configv2.ApisixPluginConfig) (*translation.TranslateContext, error)

	TranslateRouteMatchExprs(nginxVars []configv2.ApisixRouteHTTPMatchExpr) ([][]apisixv1.StringOrSlice, error)
}

func NewApisixTranslator(serviceLister listerscorev1.ServiceLister,
	secretLister listerscorev1.SecretLister,
	commonTranslator translation.Translator) ApisixTranslator {
	t := &translator{
		ServiceLister: serviceLister,
		SecretLister:  secretLister,
		Translator:    commonTranslator,
	}

	return t
}
