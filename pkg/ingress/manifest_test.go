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
	assert.Equal(t, "1", added[0].ID)
	assert.Equal(t, "3", added[1].ID)
	assert.Equal(t, []string{"POST"}, added[1].Methods)

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
	assert.Equal(t, "2", deleted[0].ID)
	assert.Equal(t, "3", deleted[1].ID)
	assert.Equal(t, []string{"POST", "PUT"}, deleted[1].Methods)

	added, updated, deleted = diffRoutes(olds, news)
	assert.Len(t, added, 1)
	assert.Equal(t, "1", added[0].ID)
	assert.Len(t, updated, 1)
	assert.Equal(t, "3", updated[0].ID)
	assert.Equal(t, []string{"POST"}, updated[0].Methods)
	assert.Len(t, deleted, 1)
	assert.Equal(t, "2", deleted[0].ID)
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
	assert.Equal(t, "1", added[0].ID)
	assert.Equal(t, "3", added[1].ID)
	assert.Equal(t, int32(8080), added[1].ServerPort)

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
	assert.Equal(t, "2", deleted[0].ID)
	assert.Equal(t, "3", deleted[1].ID)
	assert.Equal(t, int32(8081), deleted[1].ServerPort)

	added, updated, deleted = diffStreamRoutes(olds, news)
	assert.Len(t, added, 1)
	assert.Equal(t, "1", added[0].ID)
	assert.Len(t, updated, 1)
	assert.Equal(t, "3", updated[0].ID)
	assert.Equal(t, int32(8080), updated[0].ServerPort)
	assert.Len(t, deleted, 1)
	assert.Equal(t, "2", deleted[0].ID)
}

func TestDiffUpstreams(t *testing.T) {
	retries := 3
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
			Retries: &retries,
		},
	}
	added, updated, deleted := diffUpstreams(nil, news)
	assert.Nil(t, updated)
	assert.Nil(t, deleted)
	assert.Len(t, added, 2)
	assert.Equal(t, "1", added[0].ID)
	assert.Equal(t, "3", added[1].ID)
	assert.Equal(t, 3, *added[1].Retries)

	retries1 := 5
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
			Retries: &retries1,
			Timeout: &apisixv1.UpstreamTimeout{
				Connect: 10,
			},
		},
	}
	added, updated, deleted = diffUpstreams(olds, nil)
	assert.Nil(t, updated)
	assert.Nil(t, added)
	assert.Len(t, deleted, 2)
	assert.Equal(t, "2", deleted[0].ID)
	assert.Equal(t, "3", deleted[1].ID)
	assert.Equal(t, 5, *deleted[1].Retries)
	assert.Equal(t, 10, deleted[1].Timeout.Connect)

	added, updated, deleted = diffUpstreams(olds, news)
	assert.Len(t, added, 1)
	assert.Equal(t, "1", added[0].ID)
	assert.Len(t, updated, 1)
	assert.Equal(t, "3", updated[0].ID)
	assert.Nil(t, updated[0].Timeout)
	assert.Equal(t, 3, *updated[0].Retries)
	assert.Len(t, deleted, 1)
	assert.Equal(t, "2", deleted[0].ID)
}

func TestDiffPluginConfigs(t *testing.T) {
	news := []*apisixv1.PluginConfig{
		{
			Metadata: apisixv1.Metadata{
				ID: "1",
			},
		},
		{
			Metadata: apisixv1.Metadata{
				ID: "3",
			},
			Plugins: map[string]interface{}{
				"key-1": 123456,
			},
		},
	}
	added, updated, deleted := diffPluginConfigs(nil, news)
	assert.Nil(t, updated)
	assert.Nil(t, deleted)
	assert.Len(t, added, 2)
	assert.Equal(t, "1", added[0].ID)
	assert.Equal(t, "3", added[1].ID)
	assert.Equal(t, news[1].Plugins, added[1].Plugins)

	olds := []*apisixv1.PluginConfig{
		{
			Metadata: apisixv1.Metadata{
				ID: "2",
			},
		},
		{
			Metadata: apisixv1.Metadata{
				ID: "3",
			},
			Plugins: map[string]interface{}{
				"key-1": 123456789,
				"key-2": map[string][]string{
					"whitelist": {
						"127.0.0.0/24",
						"113.74.26.106",
					},
				},
			},
		},
	}
	added, updated, deleted = diffPluginConfigs(olds, nil)
	assert.Nil(t, updated)
	assert.Nil(t, added)
	assert.Len(t, deleted, 2)
	assert.Equal(t, "2", deleted[0].ID)
	assert.Equal(t, "3", deleted[1].ID)
	assert.Equal(t, olds[1].Plugins, deleted[1].Plugins)

	added, updated, deleted = diffPluginConfigs(olds, news)
	assert.Len(t, added, 1)
	assert.Equal(t, "1", added[0].ID)
	assert.Len(t, updated, 1)
	assert.Equal(t, "3", updated[0].ID)
	assert.Len(t, updated[0].Plugins, 1)
	assert.Len(t, deleted, 1)
	assert.Equal(t, "2", deleted[0].ID)
}

func TestManifestDiff(t *testing.T) {
	retries := 2
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
				Retries: &retries,
			},
		},
		pluginConfigs: []*apisixv1.PluginConfig{
			{
				Metadata: apisixv1.Metadata{
					ID: "5",
				},
				Plugins: map[string]interface{}{
					"key-1": 123456789,
					"key-2": map[string][]string{
						"whitelist": {
							"127.0.0.0/24",
							"113.74.26.106",
						},
					},
				},
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
	assert.Equal(t, "1", added.routes[0].ID)
	assert.Len(t, added.upstreams, 1)
	assert.Equal(t, "4", added.upstreams[0].ID)
	assert.Len(t, added.pluginConfigs, 1)
	assert.Equal(t, "5", added.pluginConfigs[0].ID)

	assert.Len(t, updated.routes, 1)
	assert.Equal(t, "3", updated.routes[0].ID)
	assert.Equal(t, []string{"GET"}, updated.routes[0].Methods)
	assert.Nil(t, updated.upstreams)
	assert.Nil(t, updated.pluginConfigs)

	assert.Len(t, deleted.routes, 1)
	assert.Equal(t, "2", deleted.routes[0].ID)
	assert.Nil(t, updated.upstreams)
	assert.Nil(t, updated.pluginConfigs)
}
