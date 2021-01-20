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

package cache

import v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"

// Cache defines the necessary behaviors that the cache object should have.
// Note this interface is for APISIX, not for generic purpose, it supports
// standard APISIX resources, i.e. Route, Upstream, Service and SSL.
// Cache implementations should copy the target objects before/after read/write
// operations for the sake of avoiding data corrupted by other writers.
type Cache interface {
	// InsertRoute adds or updates route to cache.
	InsertRoute(*v1.Route) error
	// InsertService adds or updates service to cache.
	InsertService(*v1.Service) error
	// InsertSSL adds or updates ssl to cache.
	InsertSSL(*v1.Ssl) error
	// InsertUpstream adds or updates upstream to cache.
	InsertUpstream(*v1.Upstream) error

	// GetRoute finds the route from cache according to the primary index.
	GetRoute(string) (*v1.Route, error)
	// GetService finds the service from cache according to the primary index.
	GetService(string) (*v1.Service, error)
	// GetSSL finds the ssl from cache according to the primary index.
	GetSSL(string) (*v1.Ssl, error)
	// GetUpstream finds the upstream from cache according to the primary index.
	GetUpstream(string) (*v1.Upstream, error)

	// ListRoutes lists all routes in cache.
	ListRoutes() ([]*v1.Route, error)
	// ListServices lists all services in cache.
	ListServices() ([]*v1.Service, error)
	// ListSSL lists all ssl objects in cache.
	ListSSL() ([]*v1.Ssl, error)
	// ListUpstreams lists all upstreams in cache.
	ListUpstreams() ([]*v1.Upstream, error)

	// DeleteRoute deletes the specified route in cache.
	DeleteRoute(*v1.Route) error
	// DeleteService deletes the specified service in cache.
	DeleteService(*v1.Service) error
	// DeleteSSL deletes the specified ssl in cache.
	DeleteSSL(*v1.Ssl) error
	// DeleteUpstream deletes the specified upstream in cache.
	DeleteUpstream(*v1.Upstream) error
}
