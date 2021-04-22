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
package ingress

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestDiffRoutes(t *testing.T) {
	news := []*apisixv1.Route{
		{
			Metadata: apisixv1.Metadata{
				ID: "1",
			},
		},
		{
			Metadata: apisixv1.Metadata{
				ID: "3",
			},
			Methods: []string{"POST"},
		},
	}
	added, updated, deleted := diffRoutes(nil, news)
	assert.Nil(t, updated)
	assert.Nil(t, deleted)
	assert.Len(t, added, 2)
	assert.Equal(t, added[0].ID, "1")
	assert.Equal(t, added[1].ID, "3")
	assert.Equal(t, added[1].Methods, []string{"POST"})

	olds := []*apisixv1.Route{
		{
			Metadata: apisixv1.Metadata{
				ID: "2",
			},
		},
		{
			Metadata: apisixv1.Metadata{
				ID: "3",
			},
			Methods: []string{"POST", "PUT"},
		},
	}
	added, updated, deleted = diffRoutes(olds, nil)
	assert.Nil(t, updated)
	assert.Nil(t, added)
	assert.Len(t, deleted, 2)
	assert.Equal(t, deleted[0].ID, "2")
	assert.Equal(t, deleted[1].ID, "3")
	assert.Equal(t, deleted[1].Methods, []string{"POST", "PUT"})

	added, updated, deleted = diffRoutes(olds, news)
	assert.Len(t, added, 1)
	assert.Equal(t, added[0].ID, "1")
	assert.Len(t, updated, 1)
	assert.Equal(t, updated[0].ID, "3")
	assert.Equal(t, updated[0].Methods, []string{"POST"})
	assert.Len(t, deleted, 1)
	assert.Equal(t, deleted[0].ID, "2")
}

func TestDiffStreamRoutes(t *testing.T) {
	news := []*apisixv1.StreamRoute{
		{
			ID: "1",
		},
		{
			ID:         "3",
			ServerPort: 8080,
		},
	}
	added, updated, deleted := diffStreamRoutes(nil, news)
	assert.Nil(t, updated)
	assert.Nil(t, deleted)
	assert.Len(t, added, 2)
	assert.Equal(t, added[0].ID, "1")
	assert.Equal(t, added[1].ID, "3")
	assert.Equal(t, added[1].ServerPort, int32(8080))

	olds := []*apisixv1.StreamRoute{
		{
			ID: "2",
		},
		{
			ID:         "3",
			ServerPort: 8081,
		},
	}
	added, updated, deleted = diffStreamRoutes(olds, nil)
	assert.Nil(t, updated)
	assert.Nil(t, added)
	assert.Len(t, deleted, 2)
	assert.Equal(t, deleted[0].ID, "2")
	assert.Equal(t, deleted[1].ID, "3")
	assert.Equal(t, deleted[1].ServerPort, int32(8081))

	added, updated, deleted = diffStreamRoutes(olds, news)
	assert.Len(t, added, 1)
	assert.Equal(t, added[0].ID, "1")
	assert.Len(t, updated, 1)
	assert.Equal(t, updated[0].ID, "3")
	assert.Equal(t, updated[0].ServerPort, int32(8080))
	assert.Len(t, deleted, 1)
	assert.Equal(t, deleted[0].ID, "2")
}

func TestDiffUpstreams(t *testing.T) {
	news := []*apisixv1.Upstream{
		{
			Metadata: apisixv1.Metadata{
				ID: "1",
			},
		},
		{
			Metadata: apisixv1.Metadata{
				ID: "3",
			},
			Retries: 3,
		},
	}
	added, updated, deleted := diffUpstreams(nil, news)
	assert.Nil(t, updated)
	assert.Nil(t, deleted)
	assert.Len(t, added, 2)
	assert.Equal(t, added[0].ID, "1")
	assert.Equal(t, added[1].ID, "3")
	assert.Equal(t, added[1].Retries, 3)

	olds := []*apisixv1.Upstream{
		{
			Metadata: apisixv1.Metadata{
				ID: "2",
			},
		},
		{
			Metadata: apisixv1.Metadata{
				ID: "3",
			},
			Retries: 5,
			Timeout: &apisixv1.UpstreamTimeout{
				Connect: 10,
			},
		},
	}
	added, updated, deleted = diffUpstreams(olds, nil)
	assert.Nil(t, updated)
	assert.Nil(t, added)
	assert.Len(t, deleted, 2)
	assert.Equal(t, deleted[0].ID, "2")
	assert.Equal(t, deleted[1].ID, "3")
	assert.Equal(t, deleted[1].Retries, 5)
	assert.Equal(t, deleted[1].Timeout.Connect, 10)

	added, updated, deleted = diffUpstreams(olds, news)
	assert.Len(t, added, 1)
	assert.Equal(t, added[0].ID, "1")
	assert.Len(t, updated, 1)
	assert.Equal(t, updated[0].ID, "3")
	assert.Nil(t, updated[0].Timeout)
	assert.Equal(t, updated[0].Retries, 3)
	assert.Len(t, deleted, 1)
	assert.Equal(t, deleted[0].ID, "2")
}

func TestManifestDiff(t *testing.T) {
	m := &manifest{
		routes: []*apisixv1.Route{
			{
				Metadata: apisixv1.Metadata{
					ID: "1",
				},
			},
			{
				Metadata: apisixv1.Metadata{
					ID: "3",
				},
				Methods: []string{"GET"},
			},
		},
		upstreams: []*apisixv1.Upstream{
			{
				Metadata: apisixv1.Metadata{
					ID: "4",
				},
				Retries: 2,
			},
		},
	}
	om := &manifest{
		routes: []*apisixv1.Route{
			{
				Metadata: apisixv1.Metadata{
					ID: "2",
				},
			},
			{
				Metadata: apisixv1.Metadata{
					ID: "3",
				},
				Methods: []string{"GET", "HEAD"},
			},
		},
	}

	added, updated, deleted := m.diff(om)
	assert.Len(t, added.routes, 1)
	assert.Equal(t, added.routes[0].ID, "1")
	assert.Len(t, added.upstreams, 1)
	assert.Equal(t, added.upstreams[0].ID, "4")

	assert.Len(t, updated.routes, 1)
	assert.Equal(t, updated.routes[0].ID, "3")
	assert.Equal(t, updated.routes[0].Methods, []string{"GET"})
	assert.Nil(t, updated.upstreams)

	assert.Len(t, deleted.routes, 1)
	assert.Equal(t, deleted.routes[0].ID, "2")
	assert.Nil(t, updated.upstreams)
}
