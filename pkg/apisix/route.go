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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/api7/ingress-controller/pkg/log"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

type routeReqBody struct {
	Desc      *string     `json:"desc,omitempty"`
	URI       *string     `json:"uri,omitempty"`
	Host      *string     `json:"host,omitempty"`
	ServiceId *string     `json:"service_id,omitempty"`
	Plugins   *v1.Plugins `json:"plugins,omitempty"`
}

type routeRespBody struct {
	Action string `json:"action"`
	Item   item   `json:"node"`
}

type routeClient struct {
	url  string
	stub *stub
}

func newRouteClient(stub *stub) Route {
	return &routeClient{
		url:  stub.baseURL + "/routes",
		stub: stub,
	}
}

func (r *routeClient) List(ctx context.Context, group string) ([]*v1.Route, error) {
	log.Infow("try to list routes in APISIX", zap.String("url", r.url))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.stub.do(req)
	if err != nil {
		return nil, err
	}
	defer drainBody(resp.Body, r.url)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var (
		routeItems listResponse
		items      []*v1.Route
	)
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&routeItems); err != nil {
		log.Errorw("failed to decode routeClient response",
			zap.String("url", r.url),
			zap.Error(err),
		)
		return nil, err
	}
	for i, item := range routeItems.Node.Items {
		if route, err := item.route(group); err != nil {
			log.Errorw("failed to convert route item",
				zap.String("url", r.url),
				zap.String("route_key", item.Key),
				zap.Error(err))

			return nil, err
		} else {
			items = append(items, route)
		}
		log.Infof("list route #%d, body: %s", i, string(item.Value))
	}

	return items, nil
}

func (r *routeClient) Create(ctx context.Context, obj *v1.Route, group string) (*v1.Route, error) {
	data, err := json.Marshal(routeReqBody{
		Desc:      obj.Name,
		URI:       obj.Path,
		Host:      obj.Host,
		ServiceId: obj.ServiceId,

		Plugins: obj.Plugins,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	resp, err := r.stub.do(req)
	if err != nil {
		return nil, err
	}

	defer drainBody(resp.Body, r.url)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var (
		routeResp routeRespBody
	)
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&routeResp); err != nil {
		return nil, err
	}

	return routeResp.Item.route(group)
}

func (r *routeClient) Delete(ctx context.Context, obj *v1.Route) error {
	log.Infof("delete route, id:%s", *obj.ID)
	url := r.url + "/" + *obj.ID
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := r.stub.do(req)
	if err != nil {
		return err
	}
	defer drainBody(resp.Body, url)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	return nil
}

func (r *routeClient) Update(ctx context.Context, obj *v1.Route) error {
	log.Infof("update route, id:%s", *obj.ID)
	body, err := json.Marshal(routeReqBody{
		Desc:      obj.Name,
		Host:      obj.Host,
		URI:       obj.Path,
		ServiceId: obj.ServiceId,
		Plugins:   obj.Plugins,
	})
	if err != nil {
		return err
	}
	url := r.url + "/" + *obj.ID
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := r.stub.do(req)
	if err != nil {
		return err
	}
	defer drainBody(resp.Body, url)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
