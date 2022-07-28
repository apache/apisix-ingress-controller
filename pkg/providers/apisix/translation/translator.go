package translation

import (
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
)

type translator struct {
	PodCache             types.PodCache
	PodLister            listerscorev1.PodLister
	EndpointLister       kube.EndpointLister
	ServiceLister        listerscorev1.ServiceLister
	ApisixUpstreamLister kube.ApisixUpstreamLister
	SecretLister         listerscorev1.SecretLister
	UseEndpointSlices    bool
	APIVersion           string

	translation.BaseTranslator
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
}

func NewApisixTranslator() (ApisixTranslator, error) {
	t := &translator{}

	return t, nil
}
