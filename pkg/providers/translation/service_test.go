package translation

import (
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"testing"
)

func TestTranslateServiceNoEndpoints(t *testing.T) {
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	epLister, _ := kube.NewEndpointListerAndInformer(informersFactory, false)
	auLister2 := v2.NewApisixUpstreamLister(cache.NewIndexer(func(obj interface{}) (out string, err error) { return }, map[string]cache.IndexFunc{}))
	auLister2beta3 := v2beta3.NewApisixUpstreamLister(cache.NewIndexer(func(obj interface{}) (out string, err error) { return }, map[string]cache.IndexFunc{}))

	tr := &translator{&TranslatorOptions{
		APIVersion:           config.ApisixV2,
		EndpointLister:       epLister,
		ApisixUpstreamLister: kube.NewApisixUpstreamLister(auLister2beta3, auLister2),
	}}

	expected := &v1.Upstream{
		Metadata: v1.Metadata{
			Desc:   "Created by apisix-ingress-controller, DO NOT modify it manually",
			Labels: map[string]string{"managed-by": "apisix-ingress-controller"},
		},
		Type:   "roundrobin",
		Nodes:  v1.UpstreamNodes{},
		Scheme: "http",
	}

	upstream, err := tr.TranslateService("test", "svc", "", 9080)
	assert.Nil(t, err)
	assert.Equal(t, expected, upstream)

	tr.APIVersion = config.ApisixV2beta3

	upstream, err = tr.TranslateService("test", "svc", "", 9080)
	assert.Nil(t, err)
	assert.Equal(t, expected, upstream)
}
