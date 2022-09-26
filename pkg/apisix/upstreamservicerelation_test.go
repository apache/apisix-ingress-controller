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
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestUpstreamServiceRelation(t *testing.T) {
	u := url.URL{}
	closedCh := make(chan struct{})
	close(closedCh)
	cache, err := cache.NewMemDBCache()
	assert.Nil(t, err)
	cli := newUpstreamServiceRelation(&cluster{
		baseURL:          u.String(),
		cli:              http.DefaultClient,
		cache:            cache,
		cacheSynced:      closedCh,
		metricsCollector: metrics.NewPrometheusCollector(),
		upstream:         &dummyUpstream{},
	})

	upsName := "default_httpbin_80"
	upsName2 := "default_httpbin_8080"
	svcName := "default_httpbin"

	err = cli.Create(context.TODO(), upsName)
	assert.Nil(t, err)

	relation, err := cli.Get(context.TODO(), svcName)
	assert.Nil(t, err)
	assert.NotNil(t, relation)
	assert.Equal(t, &v1.UpstreamServiceRelation{
		ServiceName: svcName,
		UpstreamNames: map[string]struct{}{
			upsName: {},
		},
	}, relation)

	err = cli.Create(context.TODO(), upsName2)
	assert.Nil(t, err)

	relation, err = cli.Get(context.TODO(), svcName)
	assert.Nil(t, err)
	assert.NotNil(t, relation)
	assert.Equal(t, &v1.UpstreamServiceRelation{
		ServiceName: svcName,
		UpstreamNames: map[string]struct{}{
			upsName:  {},
			upsName2: {},
		},
	}, relation)

	relations, err := cli.List(context.TODO())
	assert.Nil(t, err)
	assert.Len(t, relations, 1)
	assert.Equal(t, &v1.UpstreamServiceRelation{
		ServiceName: svcName,
		UpstreamNames: map[string]struct{}{
			upsName:  {},
			upsName2: {},
		},
	}, relations[0])

	err = cli.Delete(context.TODO(), svcName)
	assert.Nil(t, err)
	relations, err = cli.List(context.TODO())
	assert.Nil(t, err)
	assert.Len(t, relations, 0)
}

func TestUpstreamRelatoinClient(t *testing.T) {
	srv := runFakeUpstreamSrv(t)
	defer func() {
		assert.Nil(t, srv.Shutdown(context.Background()))
	}()

	cache, err := cache.NewMemDBCache()
	assert.Nil(t, err)
	u := url.URL{
		Scheme: "http",
		Host:   srv.Addr,
		Path:   "/apisix/admin",
	}
	closedCh := make(chan struct{})
	clu := &cluster{
		baseURL:          u.String(),
		cli:              http.DefaultClient,
		cache:            cache,
		cacheSynced:      closedCh,
		metricsCollector: metrics.NewPrometheusCollector(),
	}
	close(closedCh)
	relationCli := newUpstreamServiceRelation(clu)
	clu.upstreamServiceRelation = relationCli
	cli := newUpstreamClient(clu)
	clu.upstream = cli
	relationCli.cluster = clu

	// Create
	key := "upstreams/abc"
	lbType := "roundrobin"
	upsName := "default_httpbin_80"
	upsName2 := "default_httpbin_8080"
	svcName := "default_httpbin"
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
			Name: upsName,
		},
		Type:  lbType,
		Key:   key,
		Nodes: nodes,
	})
	assert.Nil(t, err)
	assert.Equal(t, "1", obj.ID)
	relations, err := relationCli.List(context.TODO())
	assert.Nil(t, err)
	assert.Len(t, relations, 1)
	assert.Equal(t, &v1.UpstreamServiceRelation{
		ServiceName: svcName,
		UpstreamNames: map[string]struct{}{
			upsName: {},
		},
	}, relations[0])

	id2 := "2"
	obj, err = cli.Create(context.TODO(), &v1.Upstream{
		Metadata: v1.Metadata{
			ID:   id2,
			Name: upsName2,
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
	relations, err = relationCli.List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, relations, 1)
	assert.Equal(t, &v1.UpstreamServiceRelation{
		ServiceName: svcName,
		UpstreamNames: map[string]struct{}{
			upsName:  {},
			upsName2: {},
		},
	}, relations[0])

	err = relationCli.Delete(context.Background(), svcName)
	assert.Nil(t, err)
	objs, err = clu.Upstream().List(context.Background())
	assert.Nil(t, err)
	assert.Len(t, objs, 2)
	assert.Equal(t, "1", objs[0].ID)
	assert.Equal(t, "2", objs[1].ID)
}
