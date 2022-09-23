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

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/nettest"

	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type fakeAPISIXUpstreamSrv struct {
	upstream map[string]json.RawMessage
}

func (srv *fakeAPISIXUpstreamSrv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if !strings.HasPrefix(r.URL.Path, "/apisix/admin/upstreams") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet {
		resp := fakeListResp{
			Count: strconv.Itoa(len(srv.upstream)),
			Node: fakeNode{
				Key: "/apisix/upstreams",
			},
		}
		var keys []string
		for key := range srv.upstream {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			resp.Node.Items = append(resp.Node.Items, fakeItem{
				Key:   key,
				Value: srv.upstream[key],
			})
		}
		w.WriteHeader(http.StatusOK)
		data, _ := json.Marshal(resp)
		_, _ = w.Write(data)
		return
	}

	if r.Method == http.MethodDelete {
		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/upstreams/")
		id = "/apisix/upstreams/" + id
		code := http.StatusNotFound
		if _, ok := srv.upstream[id]; ok {
			delete(srv.upstream, id)
			code = http.StatusOK
		}
		w.WriteHeader(code)
	}

	if r.Method == http.MethodPut {
		paths := strings.Split(r.URL.Path, "/")
		key := fmt.Sprintf("/apisix/upstreams/%s", paths[len(paths)-1])
		data, _ := ioutil.ReadAll(r.Body)
		srv.upstream[key] = data
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
		id := strings.TrimPrefix(r.URL.Path, "/apisix/admin/upstreams/")
		id = "/apisix/upstreams/" + id
		if _, ok := srv.upstream[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		data, _ := ioutil.ReadAll(r.Body)
		srv.upstream[id] = data

		w.WriteHeader(http.StatusOK)
		output := fmt.Sprintf(`{"action": "compareAndSwap", "node": {"key": "%s", "value": %s}}`, id, string(data))
		_, _ = w.Write([]byte(output))
		return
	}
}

func runFakeUpstreamSrv(t *testing.T) *http.Server {
	srv := &fakeAPISIXUpstreamSrv{
		upstream: make(map[string]json.RawMessage),
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

func TestUpstreamClient(t *testing.T) {
	srv := runFakeUpstreamSrv(t)
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
	cli := newUpstreamClient(&cluster{
		baseURL:                 u.String(),
		cli:                     http.DefaultClient,
		cache:                   &dummyCache{},
		cacheSynced:             closedCh,
		metricsCollector:        metrics.NewPrometheusCollector(),
		upstreamServiceRelation: &dummyUpstreamServiceRelation{},
	})

	// Create
	key := "upstream/abc"
	lbType := "roundrobin"
	name := "test"
	ip := "10.0.11.153"
	port := 15006
	weight := 100
	nodes := v1.UpstreamNodes{
		{
			Host:   ip,
			Port:   port,
			Weight: weight,
		},
	}

	obj, err := cli.Create(context.TODO(), &v1.Upstream{
		Metadata: v1.Metadata{
			ID:   "1",
			Name: name,
		},
		Type:  lbType,
		Key:   key,
		Nodes: nodes,
	})
	assert.Nil(t, err)
	assert.Equal(t, "1", obj.ID)

	id2 := "2"
	obj, err = cli.Create(context.TODO(), &v1.Upstream{
		Metadata: v1.Metadata{
			ID:   id2,
			Name: name,
		},
		Type:  lbType,
		Key:   key,
		Nodes: nodes,
	})
	assert.Nil(t, err)
	assert.Equal(t, "2", obj.ID)

	// List
	objs, err := cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 2)
	assert.Equal(t, "1", objs[0].ID)
	assert.Equal(t, "2", objs[1].ID)

	// Delete then List
	assert.Nil(t, cli.Delete(context.Background(), objs[0]))
	objs, err = cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 1)
	assert.Equal(t, "2", objs[0].ID)

	// Patch then List
	_, err = cli.Update(context.Background(), &v1.Upstream{
		Metadata: v1.Metadata{
			ID:   "2",
			Name: name,
		},
		Type:  "chash",
		Key:   key,
		Nodes: nodes,
	})
	assert.Nil(t, err)
	objs, err = cli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 1)
	assert.Equal(t, "2", objs[0].ID)
	assert.Equal(t, "chash", objs[0].Type)
}
