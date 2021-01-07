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

package apisix

import (
	"context"

	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

type nonExistentCluster struct {
	embedDummyResourceImplementer
}

func newNonExistentCluster() *nonExistentCluster {
	return &nonExistentCluster{
		embedDummyResourceImplementer{
			route:    &dummyRoute{},
			ssl:      &dummySSL{},
			service:  &dummyService{},
			upstream: &dummyUpstream{},
		},
	}
}

type embedDummyResourceImplementer struct {
	route    Route
	ssl      SSL
	upstream Upstream
	service  Service
}

type dummyRoute struct{}

func (f *dummyRoute) List(_ context.Context) ([]*v1.Route, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyRoute) Create(_ context.Context, _ *v1.Route) (*v1.Route, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyRoute) Delete(_ context.Context, _ *v1.Route) error {
	return ErrClusterNotExist
}

func (f *dummyRoute) Update(_ context.Context, _ *v1.Route) (*v1.Route, error) {
	return nil, ErrClusterNotExist
}

type dummySSL struct{}

func (f *dummySSL) List(_ context.Context) ([]*v1.Ssl, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySSL) Create(_ context.Context, _ *v1.Ssl) (*v1.Ssl, error) {
	return nil, ErrClusterNotExist
}

func (f *dummySSL) Delete(_ context.Context, _ *v1.Ssl) error {
	return ErrClusterNotExist
}

func (f *dummySSL) Update(_ context.Context, _ *v1.Ssl) (*v1.Ssl, error) {
	return nil, ErrClusterNotExist
}

type dummyUpstream struct{}

func (f *dummyUpstream) List(_ context.Context) ([]*v1.Upstream, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyUpstream) Create(_ context.Context, _ *v1.Upstream) (*v1.Upstream, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyUpstream) Delete(_ context.Context, _ *v1.Upstream) error {
	return ErrClusterNotExist
}

func (f *dummyUpstream) Update(_ context.Context, _ *v1.Upstream) (*v1.Upstream, error) {
	return nil, ErrClusterNotExist
}

type dummyService struct{}

func (f *dummyService) List(_ context.Context) ([]*v1.Service, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyService) Create(_ context.Context, _ *v1.Service) (*v1.Service, error) {
	return nil, ErrClusterNotExist
}

func (f *dummyService) Delete(_ context.Context, _ *v1.Service) error {
	return ErrClusterNotExist
}

func (f *dummyService) Update(_ context.Context, _ *v1.Service) (*v1.Service, error) {
	return nil, ErrClusterNotExist
}

func (nc *nonExistentCluster) Route() Route {
	return nc.route
}

func (nc *nonExistentCluster) SSL() SSL {
	return nc.ssl
}

func (nc *nonExistentCluster) Service() Service {
	return nc.service
}

func (nc *nonExistentCluster) Upstream() Upstream {
	return nc.upstream
}
