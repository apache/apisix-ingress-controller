// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cache

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
)

func TestDeleteAllRemovesPluginMetadata(t *testing.T) {
	store := NewStore(logr.Discard())
	const name = "GatewayProxy/default/gp-a"

	require.NoError(t, store.Insert(name,
		[]string{adctypes.TypeService, adctypes.TypePluginMetadata},
		&adctypes.Resources{
			Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "service-a"}}},
			PluginMetadata: adctypes.PluginMetadata{
				"prometheus": map[string]any{"prefer_name": true},
			},
		}, nil))

	require.NoError(t, store.Delete(name, nil, nil))
	_, err := store.ListGlobalRules(name)
	require.Error(t, err, "the per-GatewayProxy cache must be removed")

	// Recreating the same cache key must not resurrect metadata from its previous
	// lifecycle.
	require.NoError(t, store.Insert(name, []string{adctypes.TypeService},
		&adctypes.Resources{
			Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "service-b"}}},
		}, nil))
	resources, err := store.GetResources(name)
	require.NoError(t, err)
	assert.Empty(t, resources.PluginMetadata)
}
