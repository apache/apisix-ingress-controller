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

package scaffold

import (
	"context"
	"encoding/json"

	"github.com/gavv/httpexpect/v2"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
)

// standaloneDataplaneResource implements DataplaneResource by reading straight off
// apisix-standalone's own Admin API. adc's plain CLI has no --backend value for
// apisix-standalone -- that mode only exists inside the ingress-server's /sync and
// /validate task bodies -- so `adc dump` can't be used here the way it is for apisix
// and api7ee.
type standaloneDataplaneResource struct {
	client   *httpexpect.Expect
	adminKey string
}

func newStandaloneDataplaneResource(client *httpexpect.Expect, adminKey string) DataplaneResource {
	return &standaloneDataplaneResource{client: client, adminKey: adminKey}
}

// standaloneConfig mirrors the document GET /apisix/admin/configs returns: every
// resource type in its own flat top-level array, cross-referenced by service_id /
// upstream_id, rather than nested the way adc's own document model is. Most fields
// already share adc's json tags, so they decode directly into adc's types.
type standaloneConfig struct {
	Routes       []*adctypes.Route       `json:"routes"`
	Services     []*adctypes.Service     `json:"services"`
	Upstreams    []*adctypes.Upstream    `json:"upstreams"`
	SSLs         []*standaloneSSL        `json:"ssls"`
	StreamRoutes []*adctypes.StreamRoute `json:"stream_routes"`
	Consumers    []*adctypes.Consumer    `json:"consumers"`
}

// standaloneCrossRefs decodes the service_id / upstream_id links the flat document
// carries instead of nesting. adc's Route/Service/StreamRoute types have no field for
// them, so they're decoded separately from the same bytes and used only to rebuild the
// nesting DataplaneResource callers expect (e.g. Service().List()[0].Upstream).
type standaloneCrossRefs struct {
	Routes []struct {
		ServiceID string `json:"service_id"`
	} `json:"routes"`
	Services []struct {
		UpstreamID string `json:"upstream_id"`
	} `json:"services"`
	StreamRoutes []struct {
		ServiceID string `json:"service_id"`
	} `json:"stream_routes"`
}

// standaloneSSL mirrors APISIX's native ssl resource, which keeps a single cert/key
// pair (or a parallel certs/keys pair for multiple SNIs) instead of adc's Certificates
// list.
type standaloneSSL struct {
	adctypes.SSL

	Cert  string   `json:"cert"`
	Key   string   `json:"key"`
	Certs []string `json:"certs"`
	Keys  []string `json:"keys"`
}

func (s *standaloneSSL) toADC() *adctypes.SSL {
	out := s.SSL
	switch {
	case s.Cert != "":
		out.Certificates = []adctypes.Certificate{{Certificate: s.Cert, Key: s.Key}}
	case len(s.Certs) > 0:
		out.Certificates = make([]adctypes.Certificate, 0, len(s.Certs))
		for i, cert := range s.Certs {
			var key string
			if i < len(s.Keys) {
				key = s.Keys[i]
			}
			out.Certificates = append(out.Certificates, adctypes.Certificate{Certificate: cert, Key: key})
		}
	}
	return &out
}

// dump fetches the current config and decodes it twice from the same bytes: once into
// adc's own types, once into the cross-reference IDs those types don't carry. Both
// decodes walk the same arrays in the same order, so entries at the same index refer to
// the same resource.
func (r *standaloneDataplaneResource) dump(ctx context.Context) (*standaloneConfig, *standaloneCrossRefs, error) {
	reporter := &ErrorReporter{}
	resp := r.client.GET("/apisix/admin/configs").
		WithContext(ctx).
		WithHeader("X-API-KEY", r.adminKey).
		WithReporter(reporter).
		Expect()
	if err := reporter.Err(); err != nil {
		return nil, nil, err
	}
	body := []byte(resp.Body().Raw())

	var cfg standaloneConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, nil, err
	}
	var refs standaloneCrossRefs
	if err := json.Unmarshal(body, &refs); err != nil {
		return nil, nil, err
	}
	return &cfg, &refs, nil
}

// withNesting rebuilds the Service -> Upstream/Routes/StreamRoutes nesting from the
// cross-reference IDs, matching what callers of Service().List() expect.
func withNesting(cfg *standaloneConfig, refs *standaloneCrossRefs) []*adctypes.Service {
	upstreamByID := make(map[string]*adctypes.Upstream, len(cfg.Upstreams))
	for _, u := range cfg.Upstreams {
		upstreamByID[u.ID] = u
	}

	serviceByID := make(map[string]*adctypes.Service, len(cfg.Services))
	for i, svc := range cfg.Services {
		if i < len(refs.Services) {
			svc.Upstream = upstreamByID[refs.Services[i].UpstreamID]
		}
		serviceByID[svc.ID] = svc
	}

	for i, route := range cfg.Routes {
		if i >= len(refs.Routes) {
			continue
		}
		if svc := serviceByID[refs.Routes[i].ServiceID]; svc != nil {
			svc.Routes = append(svc.Routes, route)
		}
	}
	for i, sr := range cfg.StreamRoutes {
		if i >= len(refs.StreamRoutes) {
			continue
		}
		if svc := serviceByID[refs.StreamRoutes[i].ServiceID]; svc != nil {
			svc.StreamRoutes = append(svc.StreamRoutes, sr)
		}
	}

	return cfg.Services
}

func (r *standaloneDataplaneResource) Route() RouteResource     { return standaloneRouteResource{r} }
func (r *standaloneDataplaneResource) Service() ServiceResource { return standaloneServiceResource{r} }
func (r *standaloneDataplaneResource) SSL() SSLResource         { return standaloneSSLResource{r} }
func (r *standaloneDataplaneResource) Upstream() UpstreamResource {
	return standaloneUpstreamResource{r}
}
func (r *standaloneDataplaneResource) Consumer() ConsumerResource {
	return standaloneConsumerResource{r}
}

type standaloneRouteResource struct{ *standaloneDataplaneResource }

func (r standaloneRouteResource) List(ctx context.Context) ([]*adctypes.Route, error) {
	cfg, _, err := r.dump(ctx)
	if err != nil {
		return nil, err
	}
	return cfg.Routes, nil
}

type standaloneServiceResource struct{ *standaloneDataplaneResource }

func (r standaloneServiceResource) List(ctx context.Context) ([]*adctypes.Service, error) {
	cfg, refs, err := r.dump(ctx)
	if err != nil {
		return nil, err
	}
	return withNesting(cfg, refs), nil
}

type standaloneUpstreamResource struct{ *standaloneDataplaneResource }

func (r standaloneUpstreamResource) List(ctx context.Context) ([]*adctypes.Upstream, error) {
	cfg, _, err := r.dump(ctx)
	if err != nil {
		return nil, err
	}
	return cfg.Upstreams, nil
}

type standaloneSSLResource struct{ *standaloneDataplaneResource }

func (r standaloneSSLResource) List(ctx context.Context) ([]*adctypes.SSL, error) {
	cfg, _, err := r.dump(ctx)
	if err != nil {
		return nil, err
	}
	ssls := make([]*adctypes.SSL, 0, len(cfg.SSLs))
	for _, s := range cfg.SSLs {
		ssls = append(ssls, s.toADC())
	}
	return ssls, nil
}

type standaloneConsumerResource struct{ *standaloneDataplaneResource }

func (r standaloneConsumerResource) List(ctx context.Context) ([]*adctypes.Consumer, error) {
	cfg, _, err := r.dump(ctx)
	if err != nil {
		return nil, err
	}
	return cfg.Consumers, nil
}
