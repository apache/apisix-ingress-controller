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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/nettest"

	"github.com/stretchr/testify/assert"

	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

type fakeAPISIXRouteSrv struct {
	id    int
	route map[string]json.RawMessage
}

type fakeListResp struct {
	Count string   `json:"count"`
	Node  fakeNode `json:"node"`
}

type fakeCreateResp struct {
	Action string   `json:"action"`
	Node   fakeItem `json:"node"`
}

type fakeNode struct {
	Key   string     `json:"key"`
	Items []fakeItem `json:"nodes"`
}

type fakeItem struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

func (srv *fakeAPISIXRouteSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if !strings.HasPrefix(r.URL.Path, "/apisix/admin/routes") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet {
		resp := fakeListResp{
			Count: strconv.Itoa(len(srv.route)),
			Node: fakeNode{
				Key: "/apisix/routes",
			},
		}
		var keys []string
		for key := range srv.route {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			resp.Node.Items = append(resp.Node.Items, fakeItem{
				Key:   key,
				Value: srv.route[key],
			})
		}
		w.WriteHeader(http.StatusOK)
		data, _ := json.Marshal(resp)
		_, _ = w.Write(data)
		return
	}

	if r.Method == http.MethodDelete {
		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/routes/")
		id = "/apisix/routes/" + id
		code := http.StatusNotFound
		if _, ok := srv.route[id]; ok {
			delete(srv.route, id)
			code = http.StatusOK
		}
		w.WriteHeader(code)
	}

	if r.Method == http.MethodPost {
		srv.id++
		key := fmt.Sprintf("/apisix/routes/%d", srv.id)
		data, _ := ioutil.ReadAll(r.Body)
		srv.route[key] = data
		w.WriteHeader(http.StatusCreated)
		resp := fakeCreateResp{
			Action: "create",
			Node: fakeItem{
				Key:   key,
				Value: json.RawMessage(data),
			},
		}
		data, _ = json.Marshal(resp)
		_, _ = w.Write(data)
		return
	}

	if r.Method == http.MethodPatch {
		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/routes/")
		id = "/apisix/routes/" + id
		if _, ok := srv.route[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		data, _ := ioutil.ReadAll(r.Body)
		srv.route[id] = data

		w.WriteHeader(http.StatusOK)
		output := fmt.Sprintf(`{"action": "compareAndSwap", "node": {"key": "%s", "value": %s}}`, id, string(data))
		_, _ = w.Write([]byte(output))
		return
	}
}

func runFakeRouteSrv(t *testing.T) *http.Server {
	srv := &fakeAPISIXRouteSrv{
		id:    0,
		route: make(map[string]json.RawMessage),
	}

	ln, _ := nettest.NewLocalListener("tcp")

	httpSrv := &http.Server{
		Addr:    ln.Addr().String(),
		Handler: srv,
	}

	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			t.Errorf("failed to run http server: %s", err)
		}
	}()

	return httpSrv
}

func TestRouteClient(t *testing.T) {
	srv := runFakeRouteSrv(t)
	defer func() {
		assert.Nil(t, srv.Shutdown(context.Background()))
	}()

	u := url.URL{
		Scheme: "http",
		Host:   srv.Addr,
		Path:   "/apisix/admin",
	}

	closedCh := make(chan struct{})
	close(closedCh)
	cli := newRouteClient(&cluster{
		baseURL:    u.String(),
		cli:        http.DefaultClient,
		cache:      &dummyCache{},
		cacheReady: closedCh,
	})

	// Create
	id := "111"
	host := "www.foo.com"
	uri := "/bar"
	name := "test"
	obj, err := cli.Create(context.Background(), &v1.Route{
		Host:       &host,
		Path:       &uri,
		Name:       &name,
		ServiceId:  &id,
		UpstreamId: &id,
	})
	assert.Nil(t, err)
	assert.Equal(t, *obj.ID, "1")

	obj, err = cli.Create(context.Background(), &v1.Route{
		Host:       &host,
		Path:       &uri,
		Name:       &name,
		ServiceId:  &id,
		UpstreamId: &id,
	})
	assert.Nil(t, err)
	assert.Equal(t, *obj.ID, "2")

	// List
	objs, err := cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 2)
	assert.Equal(t, *objs[0].ID, "1")
	assert.Equal(t, *objs[1].ID, "2")

	// Delete then List
	assert.Nil(t, cli.Delete(context.Background(), objs[0]))
	objs, err = cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 1)
	assert.Equal(t, "2", *objs[0].ID)

	// Patch then List
	id = "112"
	objId := "2"
	_, err = cli.Update(context.Background(), &v1.Route{
		ID:         &objId,
		Host:       &host,
		Path:       &uri,
		Name:       &name,
		ServiceId:  &id,
		UpstreamId: &id,
	})
	assert.Nil(t, err)
	objs, err = cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 1)
	assert.Equal(t, "2", *objs[0].ID)
	assert.Equal(t, "112", *objs[0].ServiceId)
}
