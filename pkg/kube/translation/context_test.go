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
	ctx := &TranslateContext{
		upstreamMap: make(map[string]struct{}),
	}
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
	ctx.addRoute(r1)
	ctx.addRoute(r2)
	ctx.addStreamRoute(sr1)
	ctx.addStreamRoute(sr2)
	ctx.addUpstream(u1)
	ctx.addUpstream(u2)

	assert.Len(t, ctx.Routes, 2)
	assert.Len(t, ctx.StreamRoutes, 2)
	assert.Len(t, ctx.Upstreams, 1)

	assert.Equal(t, r1, ctx.Routes[0])
	assert.Equal(t, r2, ctx.Routes[1])
	assert.Equal(t, sr1, ctx.StreamRoutes[0])
	assert.Equal(t, sr2, ctx.StreamRoutes[1])
	assert.Equal(t, u1, ctx.Upstreams[0])

	assert.Equal(t, true, ctx.checkUpstreamExist("aaa"))
	assert.Equal(t, false, ctx.checkUpstreamExist("bbb"))
}
