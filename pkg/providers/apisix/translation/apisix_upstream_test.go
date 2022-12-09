package translation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestTranslateApisixUpstreamExternalNodesDomainType(t *testing.T) {
	tr := &translator{}
	defaultPort := 80
	defaultWeight := 80
	specifiedPort := 8080
	testCases := map[*v2.ApisixUpstream][]apisixv1.UpstreamNode{
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name:   "domain.foobar.com",
					Type:   "Domain",
					Weight: &defaultWeight,
					Port:   &defaultPort,
				}},
			},
		}: {{
			Host:   "domain.foobar.com",
			Port:   defaultPort,
			Weight: defaultWeight,
		}},
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name:   "domain.foobar.com",
					Type:   "Domain",
					Weight: &defaultWeight,
					Port:   &specifiedPort,
				}},
			},
		}: {{
			Host:   "domain.foobar.com",
			Port:   specifiedPort,
			Weight: defaultWeight,
		}},
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name:   "domain.foobar.com",
					Type:   "Domain",
					Weight: &defaultWeight,
				}},
			},
		}: {{
			Host:   "domain.foobar.com",
			Port:   defaultPort,
			Weight: defaultWeight,
		}},
	}
	for k, v := range testCases {
		result, _ := tr.TranslateApisixUpstreamExternalNodes(k)
		assert.Equal(t, v, result)
	}

}
