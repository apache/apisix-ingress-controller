// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package apisix

import (
	"context"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var (
	_ StreamRoute = (*noopClient)(nil)
)

type noopClient struct {
}

func (r *noopClient) Get(ctx context.Context, name string) (*v1.StreamRoute, error) {
	return nil, nil
}

func (r *noopClient) List(ctx context.Context) ([]*v1.StreamRoute, error) {
	return nil, nil
}

func (r *noopClient) Create(ctx context.Context, obj *v1.StreamRoute, shouldCompare bool) (*v1.StreamRoute, error) {
	return nil, nil
}

func (r *noopClient) Delete(ctx context.Context, obj *v1.StreamRoute) error {
	return nil
}

func (r *noopClient) Update(ctx context.Context, obj *v1.StreamRoute, shouldCompare bool) (*v1.StreamRoute, error) {
	return nil, nil
}
