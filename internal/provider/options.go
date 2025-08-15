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

package provider

import "time"

type Option interface {
	ApplyToList(*Options)
}

type Options struct {
	SyncTimeout      time.Duration
	SyncPeriod       time.Duration
	InitSyncDelay    time.Duration
	BackendMode      string
	ResolveEndpoints bool
}

func (o *Options) ApplyToList(lo *Options) {
	if o.SyncTimeout > 0 {
		lo.SyncTimeout = o.SyncTimeout
	}
	if o.SyncPeriod > 0 {
		lo.SyncPeriod = o.SyncPeriod
	}
	if o.InitSyncDelay > 0 {
		lo.InitSyncDelay = o.InitSyncDelay
	}
	if o.BackendMode != "" {
		lo.BackendMode = o.BackendMode
	}
	if o.ResolveEndpoints {
		lo.ResolveEndpoints = o.ResolveEndpoints
	}
}

func (o *Options) ApplyOptions(opts []Option) *Options {
	for _, opt := range opts {
		opt.ApplyToList(o)
	}
	return o
}

type backendModeOption string

func (b backendModeOption) ApplyToList(o *Options) {
	o.BackendMode = string(b)
}

func WithBackendMode(mode string) Option {
	return backendModeOption(mode)
}

type resolveEndpointsOption bool

func (r resolveEndpointsOption) ApplyToList(o *Options) {
	o.ResolveEndpoints = bool(r)
}

func WithResolveEndpoints() Option {
	return resolveEndpointsOption(true)
}
