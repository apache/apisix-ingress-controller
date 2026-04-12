package manager

import (
	"testing"

	netv1 "k8s.io/api/networking/v1"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	types "github.com/apache/apisix-ingress-controller/internal/types"
)

func TestAPIv2ReadinessResources(t *testing.T) {
	resources := apiV2ReadinessResources()
	seen := map[string]bool{}
	for _, resource := range resources {
		seen[types.GvkOf(resource).String()] = true
	}

	want := []string{
		types.GvkOf(&netv1.Ingress{}).String(),
		types.GvkOf(&apiv2.ApisixRoute{}).String(),
		types.GvkOf(&apiv2.ApisixGlobalRule{}).String(),
		types.GvkOf(&apiv2.ApisixTls{}).String(),
		types.GvkOf(&apiv2.ApisixConsumer{}).String(),
	}
	for _, gvk := range want {
		if !seen[gvk] {
			t.Fatalf("expected readiness resources to include %s", gvk)
		}
	}

	notExpected := []string{
		types.GvkOf(&apiv2.ApisixPluginConfig{}).String(),
		types.GvkOf(&apiv2.ApisixUpstream{}).String(),
	}
	for _, gvk := range notExpected {
		if seen[gvk] {
			t.Fatalf("expected readiness resources to exclude %s", gvk)
		}
	}
}
