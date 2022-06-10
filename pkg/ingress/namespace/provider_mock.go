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
//
package namespace

import (
	"context"
	"k8s.io/client-go/tools/cache"
)

func NewMockWatchingProvider(namespaces []string) WatchingProvider {
	return &mockWatchingProvider{
		namespaces: namespaces,
	}
}

type mockWatchingProvider struct {
	namespaces []string
}

func (c *mockWatchingProvider) Run(ctx context.Context) {
}

func (c *mockWatchingProvider) WatchingNamespaces() []string {
	return c.namespaces
}

func (c *mockWatchingProvider) IsWatchingNamespace(key string) (ok bool) {
	ns, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false
	}

	for _, namespace := range c.namespaces {
		if namespace == ns {
			return true
		}
	}
	return false
}
