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

// Package client talks to the ADC server: given a fully-prepared sync or validate
// request, it translates it to ADC's wire format, sends it once, and interprets the
// response into a typed error. It holds no bookkeeping of its own: not which Kubernetes
// resource maps to which GatewayProxy, not a GatewayProxy's current resource snapshot,
// and not whether a data plane's diff baseline can be trusted. All of that is AIC's own
// state, owned by the caller, which also owns every decision to retry.
package client

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/types"
	pkgmetrics "github.com/apache/apisix-ingress-controller/pkg/metrics"
)

type Client struct {
	executor ADCExecutor

	defaultMode string

	log logr.Logger
}

func New(log logr.Logger, defaultMode string, timeout time.Duration) (*Client, error) {
	serverURL := os.Getenv("ADC_SERVER_URL")
	if serverURL == "" {
		serverURL = defaultHTTPADCExecutorAddr
	}

	logger := log.WithName("client")
	logger.Info("ADC client initialized")

	return &Client{
		executor:    NewHTTPADCExecutor(log, serverURL, timeout),
		log:         logger,
		defaultMode: defaultMode,
	}, nil
}

// IsConfVersionRejection reports whether err is the data plane refusing a push because
// its conf_version is behind. That is the one rejection a caller can answer, by asking
// ADC to rebuild its diff baseline from the data plane (SyncInput.Config.BypassCache) and
// syncing again. This package never makes that decision; it only lets a caller recognize
// the case.
//
// It matches the field name, not the sentence. conf_version is part of the standalone
// admin API, callers send those keys themselves, so any rejection that concerns it names
// the field whatever prose APISIX wraps it in. Matching the sentence would tie this to
// prose APISIX is free to reword; matching the field only breaks if it renames the API.
func IsConfVersionRejection(err error) bool {
	return err != nil && strings.Contains(err.Error(), confVersionField)
}

// confVersionField names the monotonic version APISIX standalone keeps per resource type
// (routes_conf_version, upstreams_conf_version, ...) and refuses a push that moves back.
const confVersionField = "conf_version"

// Task is a /validate request: one Kubernetes resource's translated result, checked
// against every GatewayProxy config it could target.
type Task struct {
	Name          string
	Labels        map[string]string
	Configs       map[types.NamespacedNameKind]adctypes.Config
	ResourceTypes []string
	Resources     *adctypes.Resources
}

// MarshalLog implements logr.Marshaler so logging a Task never dumps the
// secret-bearing Resources bodies (SSL private keys, consumer credentials).
// Configs redact their own Token via Config.MarshalJSON.
func (t Task) MarshalLog() any {
	configNames := make([]string, 0, len(t.Configs))
	for _, cfg := range t.Configs {
		configNames = append(configNames, cfg.Name)
	}
	return map[string]any{
		"name":          t.Name,
		"labels":        t.Labels,
		"resourceTypes": t.ResourceTypes,
		"configs":       configNames,
		"resources":     t.Resources.MarshalLog(),
	}
}

func (c *Client) Validate(ctx context.Context, task Task) error {
	if len(task.Configs) == 0 || task.Resources == nil {
		return nil
	}

	var errs types.ADCValidationErrors
	for _, config := range task.Configs {
		if config.BackendType == "" {
			config.BackendType = c.defaultMode
		}
		if err := c.executor.Validate(ctx, config, task.Resources, task.Labels, task.ResourceTypes); err != nil {
			var validationErr types.ADCValidationError
			if errors.As(err, &validationErr) {
				errs.Errors = append(errs.Errors, validationErr)
				continue
			}
			return err
		}
	}

	if len(errs.Errors) > 0 {
		return errs
	}
	return nil
}

// SyncInput is one GatewayProxy's complete sync unit. AIC builds it entirely from its own
// bookkeeping (which resources target this config, their merged translated snapshot)
// before handing it over: this package never reaches back into AIC's state to gather
// anything itself, it only translates, sends, and interprets the response.
type SyncInput struct {
	// Name is the cacheKey: the GatewayProxy's own identity.
	Name          string
	Config        adctypes.Config
	Resources     *adctypes.Resources
	ResourceTypes []string
	Labels        map[string]string
}

// MarshalLog implements logr.Marshaler so logging a SyncInput never dumps the
// secret-bearing Resources body. Config redacts its own Token via Config.MarshalJSON.
func (in SyncInput) MarshalLog() any {
	return map[string]any{
		"name":          in.Name,
		"config":        in.Config,
		"labels":        in.Labels,
		"resourceTypes": in.ResourceTypes,
		"resources":     in.Resources.MarshalLog(),
	}
}

// Sync sends in to its data plane once and returns the parsed, typed error if the push
// failed, or nil if it succeeded. It never returns a raw HTTP status or body: every
// response ADC can send back is already interpreted by the time it gets here, into a
// types.ADCExecutionServerAddrError. The raw status this call's own metrics are labeled
// with never leaves this function.
//
// It never retries. A caller that retries (see IsConfVersionRejection) may call this more
// than once for what is, from the outside, one logical sync; this call's own duration and
// (on failure) error are recorded here regardless, so each underlying HTTP round trip
// stays individually visible, but only the caller knows when that logical sync is
// actually over, and owns whatever metric reflects that.
func (c *Client) Sync(ctx context.Context, in SyncInput) error {
	if in.Resources == nil {
		return nil
	}
	c.log.V(1).Info("syncing resources", "input", in)

	config := in.Config
	if config.BackendType == "" {
		config.BackendType = c.defaultMode
	}

	startTime := time.Now()
	statusCode, err := c.executor.Execute(ctx, config, in.Resources, in.Labels, in.ResourceTypes)

	status := adctypes.StatusSuccess
	if err != nil {
		status = "failure"
		c.log.Error(err, "failed to sync with ADC", "config", config)
		pkgmetrics.RecordClientSyncError(config.Name, statusCode)
	}
	pkgmetrics.RecordClientSyncDuration(config.Name, status, time.Since(startTime).Seconds())

	return err
}
