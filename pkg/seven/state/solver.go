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
package state

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/multierr"

	"github.com/api7/ingress-controller/pkg/apisix/cache"
	"github.com/api7/ingress-controller/pkg/log"
	"github.com/api7/ingress-controller/pkg/seven/conf"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

var UpstreamQueue chan UpstreamQueueObj

func init() {
	UpstreamQueue = make(chan UpstreamQueueObj, 500)
	go WatchUpstream()
}

func WatchUpstream() {
	for {
		uqo := <-UpstreamQueue
		SolverUpstream(uqo.Upstreams, uqo.ServiceWorkerGroup, uqo.Wg, uqo.ErrorChan)
	}
}

// Solver
func (s *ApisixCombination) Solver() (string, error) {
	// define the result notify
	timeout := 10 * time.Second
	resultChan := make(chan CRDStatus)
	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, timeout)
	go s.SyncWithGroup(ctx, "", resultChan)

	return WaitWorkerGroup("", resultChan)
}

func (s *ApisixCombination) Remove() error {
	// services
	for _, svc := range s.Services {
		var cluster string
		if svc.Group != nil {
			cluster = *svc.Group
		}
		svcInCache, err := conf.Client.Cluster(cluster).Service().Get(context.TODO(), *svc.FullName)
		if err != nil {
			if err == cache.ErrNotFound {
				log.Errorf("failed to remove service %s: %s", *svc.FullName, err)
				continue
			} else {
				return err
			}
		}
		_ = paddingService(svc, svcInCache)
		err = conf.Client.Cluster(cluster).Service().Delete(context.TODO(), svc)
		if err != nil {
			if err == cache.ErrNotFound {
				log.Errorf("failed to remove service %s: %s", *svc.FullName, err)
			} else if err == cache.ErrStillInUse {
				log.Warnf("failed to remove service %s: %s", *svc.FullName, err)
			} else {
				return err
			}
		}
	}

	// upstreams
	for _, ups := range s.Upstreams {
		var cluster string
		if ups.Group != nil {
			cluster = *ups.Group
		}
		upsInCache, err := conf.Client.Cluster(cluster).Upstream().Get(context.TODO(), *ups.FullName)
		if err != nil {
			if err == cache.ErrNotFound {
				log.Errorf("failed to remove service %s: %s", *ups.FullName, err)
				continue
			} else {
				return err
			}
		}
		_ = paddingUpstream(ups, upsInCache)
		err = conf.Client.Cluster(cluster).Upstream().Delete(context.TODO(), ups)
		if err == cache.ErrNotFound {
			log.Errorf("failed to remove upstream %s: %s", *ups.FullName, err)
		} else if err == cache.ErrStillInUse {
			log.Warnf("failed to remove upstream %s: %s", *ups.FullName, err)
		} else {
			return err
		}
	}
	return nil
}

func waitTimeout(ctx context.Context, wg *sync.WaitGroup, resultChan chan CRDStatus) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		resultChan <- CRDStatus{Id: "", Status: "success", Err: nil}
	case <-ctx.Done():
		resultChan <- CRDStatus{Id: "", Status: "failure", Err: errors.New("timeout")}
	}
}

func (s *ApisixCombination) SyncWithGroup(ctx context.Context, id string, resultChan chan CRDStatus) {
	var wg sync.WaitGroup
	count := len(s.Routes) + len(s.Services) + len(s.Upstreams)
	wg.Add(count)
	// goroutine for sync route/service/upstream
	// route
	rwg := NewRouteWorkers(ctx, s.Routes, &wg, resultChan)
	// service
	swg := NewServiceWorkers(ctx, s.Services, &rwg, &wg, resultChan)
	// upstream
	uqo := &UpstreamQueueObj{Upstreams: s.Upstreams, ServiceWorkerGroup: swg, Wg: &wg, ErrorChan: resultChan}
	uqo.AddQueue()

	waitTimeout(ctx, &wg, resultChan)
}

func WaitWorkerGroup(id string, resultChan chan CRDStatus) (string, error) {
	r := <-resultChan
	return id, r.Err
}

// UpstreamQueueObj for upstream queue
type UpstreamQueueObj struct {
	Upstreams          []*v1.Upstream
	ServiceWorkerGroup ServiceWorkerGroup
	Wg                 *sync.WaitGroup
	ErrorChan          chan CRDStatus
}

type CRDStatus struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Err    error  `json:"err"`
}

type ResourceStatus struct {
	Kind string `json:"kind"`
	Id   string `json:"id"`
	Err  error  `json:"err"`
}

// AddQueue make upstreams in order
// upstreams is group by CRD
func (uqo *UpstreamQueueObj) AddQueue() {
	UpstreamQueue <- *uqo
}

type ServiceQueueObj struct {
	Services         []*v1.Service
	RouteWorkerGroup RouteWorkerGroup
}

// Sync remove from apisix
func (rc *RouteCompare) Sync() error {
	var merr error
	for _, old := range rc.OldRoutes {
		needToDel := true
		for _, nr := range rc.NewRoutes {
			if old.Name == nr.Name {
				needToDel = false
				break
			}
		}
		if needToDel {
			var cluster string
			if old.Group != nil {
				cluster = *old.Group
			}

			// old should inject the ID.
			route, err := conf.Client.Cluster(cluster).Route().Get(context.TODO(), *old.FullName)
			if err != nil {
				if err != cache.ErrNotFound {
					merr = multierr.Append(merr, err)
				}
				continue
			}

			_ = paddingRoute(old, route)
			if err := conf.Client.Cluster(cluster).Route().Delete(context.TODO(), old); err != nil {
				log.Errorf("failed to delete route %s from APISIX: %s", *old.Name, err)
				merr = multierr.Append(merr, err)
			}
		}
	}
	return merr
}

func SyncSsl(ssl *v1.Ssl, method string) error {
	var cluster string
	if ssl.Group != nil {
		cluster = *ssl.Group
	}
	switch method {
	case Create:
		_, err := conf.Client.Cluster(cluster).SSL().Create(context.TODO(), ssl)
		return err
	case Update:
		// FIXME we don't know the full name of SSL.
		_, err := conf.Client.Cluster(cluster).SSL().Update(context.TODO(), ssl)
		return err
	case Delete:
		// FIXME we don't know the full name of SSL.
		return conf.Client.Cluster(cluster).SSL().Delete(context.TODO(), ssl)
	}
	return nil
}
