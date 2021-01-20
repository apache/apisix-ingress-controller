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

	"github.com/api7/ingress-controller/pkg/log"
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
	name     string
	baseURL  string
	adminKey string
	cli      *http.Client

	route    Route
	upstream Upstream
	service  Service
	ssl      SSL
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
	}
	c.route = newRouteClient(c)
	c.upstream = newUpstreamClient(c)
	c.service = newServiceClient(c)
	c.ssl = newSSLClient(c)

	return c, nil
}

// String exposes the client information in human readable format.
func (c *cluster) String() string {
	return fmt.Sprintf("name=%s; base_url=%s", c.name, c.baseURL)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
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
