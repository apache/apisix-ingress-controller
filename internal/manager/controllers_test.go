// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
