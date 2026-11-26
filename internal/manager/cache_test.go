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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func TestCacheOptionsWithoutSecretNamespaces(t *testing.T) {
	opts := cacheOptions(nil)
	assert.Empty(t, opts.ByObject, "Secrets are cached cluster-wide by default")
	assert.Empty(t, opts.DefaultNamespaces)
}

func TestCacheOptionsWithSecretNamespaces(t *testing.T) {
	opts := cacheOptions([]string{"ingress-apisix", "team-a"})

	assert.Empty(t, opts.DefaultNamespaces, "only the Secret cache is limited")
	require.Len(t, opts.ByObject, 1)
	for obj, byObject := range opts.ByObject {
		assert.IsType(t, &corev1.Secret{}, obj)
		assert.Equal(t, map[string]cache.Config{
			"ingress-apisix": {},
			"team-a":         {},
		}, byObject.Namespaces)
	}
}
