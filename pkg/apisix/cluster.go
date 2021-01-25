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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

const (
	_defaultTimeout = 5 * time.Second
)

var (
	// ErrClusterNotExist means a cluster doesn't exist.
	ErrClusterNotExist = errors.New("client not exist")
	// ErrDuplicatedCluster means the cluster adding request was
	// rejected since the cluster was already created.
	ErrDuplicatedCluster = errors.New("duplicated cluster")

	_errReadOnClosedResBody = errors.New("http: read on closed response body")
)

// Options contains parameters to customize APISIX client.
type ClusterOptions struct {
	Name     string
	AdminKey string
	BaseURL  string
	Timeout  time.Duration
}

type cluster struct {
	name         string
	baseURL      string
	adminKey     string
	cli          *http.Client
	cache        cache.Cache
	cacheSynced  chan struct{}
	cacheSyncErr error
	route        Route
	upstream     Upstream
	service      Service
	ssl          SSL
}

func newCluster(o *ClusterOptions) (Cluster, error) {
	if o.BaseURL == "" {
		return nil, errors.New("empty base url")
	}
	if o.Timeout == time.Duration(0) {
		o.Timeout = _defaultTimeout
	}
	o.BaseURL = strings.TrimSuffix(o.BaseURL, "/")

	c := &cluster{
		name:     o.Name,
		baseURL:  o.BaseURL,
		adminKey: o.AdminKey,
		cli: &http.Client{
			Timeout: o.Timeout,
			Transport: &http.Transport{
				ResponseHeaderTimeout: o.Timeout,
				ExpectContinueTimeout: o.Timeout,
			},
		},
		cache:       nil,
		cacheSynced: make(chan struct{}),
	}
	c.route = newRouteClient(c)
	c.upstream = newUpstreamClient(c)
	c.service = newServiceClient(c)
	c.ssl = newSSLClient(c)

	go c.syncCache()

	return c, nil
}

func (c *cluster) syncCache() {
	log.Infow("syncing cache", zap.String("cluster", c.name))
	now := time.Now()
	defer func() {
		if c.cacheSyncErr == nil {
			log.Infow("cache synced",
				zap.String("cost_time", time.Since(now).String()),
				zap.String("cluster", c.name),
			)
		} else {
			log.Errorw("failed to sync cache",
				zap.String("cost_time", time.Since(now).String()),
				zap.String("cluster", c.name),
			)
		}
	}()

	backoff := wait.Backoff{
		Duration: time.Second,
		Factor:   2,
		Steps:    6,
	}
	err := wait.ExponentialBackoff(backoff, c.syncCacheOnce)
	if err != nil {
		c.cacheSyncErr = err
	}
	close(c.cacheSynced)
}

func (c *cluster) syncCacheOnce() (bool, error) {
	dbcache, err := cache.NewMemDBCache()
	if err != nil {
		return false, err
	}
	c.cache = dbcache

	routes, err := c.route.List(context.TODO())
	if err != nil {
		log.Errorf("failed to list route in APISIX: %s", err)
		return false, err
	}
	services, err := c.service.List(context.TODO())
	if err != nil {
		log.Errorf("failed to list services in APISIX: %s", err)
		return false, err
	}
	upstreams, err := c.upstream.List(context.TODO())
	if err != nil {
		log.Errorf("failed to list upstreams in APISIX: %s", err)
		return false, err
	}
	ssl, err := c.ssl.List(context.TODO())
	if err != nil {
		log.Errorf("failed to list ssl in APISIX: %s", err)
		return false, err
	}

	for _, r := range routes {
		if err := c.cache.InsertRoute(r); err != nil {
			log.Errorw("failed to insert route to cache",
				zap.String("route", r.ID),
				zap.String("cluster", c.name),
				zap.String("error", err.Error()),
			)
			return false, err
		}
	}
	for _, s := range services {
		if err := c.cache.InsertService(s); err != nil {
			log.Errorw("failed to insert service to cache",
				zap.String("service", s.ID),
				zap.String("cluster", c.name),
				zap.String("error", err.Error()),
			)
			return false, err
		}
	}
	for _, u := range upstreams {
		if err := c.cache.InsertUpstream(u); err != nil {
			log.Errorw("failed to insert upstream to cache",
				zap.String("upstream", u.ID),
				zap.String("cluster", c.name),
				zap.String("error", err.Error()),
			)
			return false, err
		}
	}
	for _, s := range ssl {
		if err := c.cache.InsertSSL(s); err != nil {
			log.Errorw("failed to insert ssl to cache",
				zap.String("ssl", s.ID),
				zap.String("cluster", c.name),
				zap.String("error", err.Error()),
			)
			return false, err
		}
	}
	return true, nil
}

// String implements Cluster.String method.
func (c *cluster) String() string {
	return fmt.Sprintf("name=%s; base_url=%s", c.name, c.baseURL)
}

// HasSynced implements Cluster.HasSynced method.
func (c *cluster) HasSynced(ctx context.Context) error {
	if c.cacheSyncErr != nil {
		return c.cacheSyncErr
	}
	now := time.Now()
	log.Warnf("waiting cluster %s to ready, it may takes a while", c.name)
	select {
	case <-ctx.Done():
		log.Errorf("failed to wait cluster to ready: %s", ctx.Err())
		return ctx.Err()
	case <-c.cacheSynced:
		log.Warnf("cluster %s now is ready, cost time %s", c.name, time.Since(now).String())
		return nil
	}
}

// Route implements Cluster.Route method.
func (c *cluster) Route() Route {
	return c.route
}

// Upstream implements Cluster.Upstream method.
func (c *cluster) Upstream() Upstream {
	return c.upstream
}

// Service implements Cluster.Service method.
func (c *cluster) Service() Service {
	return c.service
}

// SSL implements Cluster.SSL method.
func (c *cluster) SSL() SSL {
	return c.ssl
}

func (s *cluster) applyAuth(req *http.Request) {
	if s.adminKey != "" {
		req.Header.Set("X-API-Key", s.adminKey)
	}
}

func (s *cluster) do(req *http.Request) (*http.Response, error) {
	s.applyAuth(req)
	return s.cli.Do(req)
}

func (s *cluster) getResource(ctx context.Context, url string) (*getResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer drainBody(resp.Body, url)
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, cache.ErrNotFound
		} else {
			err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
			err = multierr.Append(err, fmt.Errorf("error message: %s", readBody(resp.Body, url)))
		}
		return nil, err
	}

	var res getResponse

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *cluster) listResource(ctx context.Context, url string) (*listResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer drainBody(resp.Body, url)
	if resp.StatusCode != http.StatusOK {
		err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
		err = multierr.Append(err, fmt.Errorf("error message: %s", readBody(resp.Body, url)))
		return nil, err
	}

	var list listResponse

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&list); err != nil {
		return nil, err
	}
	return &list, nil
}

func (s *cluster) createResource(ctx context.Context, url string, body io.Reader) (*createResponse, error) {
	return s.createResourceWithMethod(ctx, url, body, http.MethodPut)
}

func (s *cluster) createResourceWithMethod(ctx context.Context, url string, body io.Reader, method string) (*createResponse, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}

	defer drainBody(resp.Body, url)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
		err = multierr.Append(err, fmt.Errorf("error message: %s", readBody(resp.Body, url)))
		return nil, err
	}

	var cr createResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

func (s *cluster) updateResource(ctx context.Context, url string, body io.Reader) (*updateResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, body)
	if err != nil {
		return nil, err
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer drainBody(resp.Body, url)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
		err = multierr.Append(err, fmt.Errorf("error message: %s", readBody(resp.Body, url)))
		return nil, err
	}
	var ur updateResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&ur); err != nil {
		return nil, err
	}
	return &ur, nil
}

func (s *cluster) deleteResource(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.do(req)
	if err != nil {
		return err
	}
	defer drainBody(resp.Body, url)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		err = multierr.Append(err, fmt.Errorf("unexpected status code %d", resp.StatusCode))
		err = multierr.Append(err, fmt.Errorf("error message: %s", readBody(resp.Body, url)))
		return err
	}
	return nil
}

// drainBody reads whole data until EOF from r, then close it.
func drainBody(r io.ReadCloser, url string) {
	_, err := io.Copy(ioutil.Discard, r)
	if err != nil {
		if err.Error() != _errReadOnClosedResBody.Error() {
			log.Warnw("failed to drain body (read)",
				zap.String("url", url),
				zap.Error(err),
			)
		}
	}

	if err := r.Close(); err != nil {
		log.Warnw("failed to drain body (close)",
			zap.String("url", url),
			zap.Error(err),
		)
	}
}

func readBody(r io.ReadCloser, url string) string {
	defer func() {
		if err := r.Close(); err != nil {
			log.Warnw("failed to close body", zap.String("url", url), zap.Error(err))
		}
	}()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		log.Warnw("failed to read body", zap.String("url", url), zap.Error(err))
		return ""
	}
	return string(data)
}
