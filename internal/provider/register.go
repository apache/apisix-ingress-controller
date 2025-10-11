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

import (
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
)

type RegisterHandler interface {
	Register(pathPrefix string, mux *http.ServeMux)
}

type RegisterFunc func(logr.Logger, status.Updater, readiness.ReadinessManager, ...Option) (Provider, error)

var providers = map[string]RegisterFunc{}

func Register(name string, registerFunc RegisterFunc) {
	providers[name] = registerFunc
}

func Get(name string) (RegisterFunc, error) {
	f, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", name)
	}
	return f, nil
}

func New(
	providerType string,
	log logr.Logger,
	updater status.Updater,
	readinesser readiness.ReadinessManager,
	opts ...Option,
) (Provider, error) {
	f, err := Get(providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %q: %w", providerType, err)
	}
	return f(log, updater, readinesser, opts...)
}
