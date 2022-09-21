// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestTranslateContext(t *testing.T) {
	ctx := DefaultEmptyTranslateContext()

	r1 := &apisix.Route{
		Metadata: apisix.Metadata{
			ID: "1",
		},
	}
	r2 := &apisix.Route{
		Metadata: apisix.Metadata{
			ID: "2",
		},
	}
	sr1 := &apisix.StreamRoute{
		ID: "1",
	}
	sr2 := &apisix.StreamRoute{
		ID: "2",
	}
	u1 := &apisix.Upstream{
		Metadata: apisix.Metadata{
			ID:   "1",
			Name: "aaa",
		},
	}
	u2 := &apisix.Upstream{
		Metadata: apisix.Metadata{
			ID:   "1",
			Name: "aaa",
		},
	}
	pc1 := &apisix.PluginConfig{
		Metadata: apisix.Metadata{
			ID:   "1",
			Name: "aaa",
		},
	}
	pc2 := &apisix.PluginConfig{
		Metadata: apisix.Metadata{
			ID:   "2",
			Name: "aaa",
		},
	}
	ctx.AddRoute(r1)
	ctx.AddRoute(r2)
	ctx.AddStreamRoute(sr1)
	ctx.AddStreamRoute(sr2)
	ctx.AddUpstream(u1)
	ctx.AddUpstream(u2)
	ctx.AddPluginConfig(pc1)
	ctx.AddPluginConfig(pc2)

	assert.Len(t, ctx.Routes, 2)
	assert.Len(t, ctx.StreamRoutes, 2)
	assert.Len(t, ctx.Upstreams, 1)
	assert.Len(t, ctx.PluginConfigs, 2)

	assert.Equal(t, r1, ctx.Routes[0])
	assert.Equal(t, r2, ctx.Routes[1])
	assert.Equal(t, sr1, ctx.StreamRoutes[0])
	assert.Equal(t, sr2, ctx.StreamRoutes[1])
	assert.Equal(t, u1, ctx.Upstreams[0])
	assert.Equal(t, pc1, ctx.PluginConfigs[0])
	assert.Equal(t, pc2, ctx.PluginConfigs[1])

	assert.Equal(t, true, ctx.CheckUpstreamExist("aaa"))
	assert.Equal(t, false, ctx.CheckUpstreamExist("bbb"))
}
