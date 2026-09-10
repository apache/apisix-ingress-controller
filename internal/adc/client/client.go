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
// request, it translates it to ADC's wire format, sends it, and interprets the response.
// It holds no bookkeeping of its own about which Kubernetes resource maps to which
// GatewayProxy, or what a GatewayProxy's current resource snapshot is -- that is AIC's own
// state, owned by the caller and handed in as input on every call.
package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
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

	// rebuiltMu guards rebuiltBaselines.
	rebuiltMu sync.Mutex
	// rebuiltBaselines holds the cacheKeys whose ADC baseline this leadership term has
	// already re-derived from the data plane. A key missing from it is synced with
	// bypassCache first. See InvalidateADCCache.
	rebuiltBaselines map[string]struct{}

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
		rebuiltBaselines: make(map[string]struct{}),
		executor:         NewHTTPADCExecutor(log, serverURL, timeout),
		log:              logger,
		defaultMode:      defaultMode,
	}, nil
}

// InvalidateADCCache forgets which ADC baselines are known to be current, so that the
// next sync of each cacheKey re-derives its baseline from the data plane.
//
// It is called on leader acquisition, which is the one moment a stale baseline can enter
// the picture. The ADC server is a sidecar that outlives the controller process: losing
// the lease terminates the manager container but not the sidecar, so what ADC holds for a
// cacheKey -- the last synced content plus the conf_version it generated -- can still be
// the snapshot this pod left behind in an earlier term, while the leader in between kept
// pushing and moved the data plane's conf_version past it. APISIX standalone requires
// those versions to be monotonic and refuses the whole configuration otherwise.
func (c *Client) InvalidateADCCache() {
	c.rebuiltMu.Lock()
	defer c.rebuiltMu.Unlock()
	clear(c.rebuiltBaselines)
}

func (c *Client) baselineIsCurrent(cacheKey string) bool {
	c.rebuiltMu.Lock()
	defer c.rebuiltMu.Unlock()
	_, ok := c.rebuiltBaselines[cacheKey]
	return ok
}

func (c *Client) markBaselineCurrent(cacheKey string) {
	c.rebuiltMu.Lock()
	defer c.rebuiltMu.Unlock()
	c.rebuiltBaselines[cacheKey] = struct{}{}
}

// isConfVersionRejection reports whether the data plane refused the push because of a
// conf_version, which is the one rejection re-deriving the baseline can answer.
//
// It matches the field name, not the sentence. conf_version is part of the standalone
// admin API -- we send those keys ourselves -- so any rejection that concerns it names it,
// whatever prose APISIX wraps it in. Matching the sentence would tie us to prose APISIX is
// free to reword; matching the field only breaks if it renames the API.
//
// This backs the safety net, not the fix. A baseline is rebuilt on leader acquisition,
// which is where staleness comes from, so if this ever stopped firing the reported bug
// would not come back with it.
func isConfVersionRejection(err error) bool {
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
// before handing it over -- this package never reaches back into AIC's state to gather
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

// Sync pushes every given SyncInput to its data plane in one sweep, and reports the
// parsed, typed error for each one that failed, keyed by its Name -- an input whose name
// is absent from the returned map genuinely succeeded. It never returns a raw HTTP status
// or body; every response ADC can send back is already interpreted by the time it gets
// here.
func (c *Client) Sync(ctx context.Context, inputs []SyncInput) (map[string]types.ADCExecutionErrors, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	c.log.V(1).Info("syncing resources", "inputs", inputs)

	failedMap := map[string]types.ADCExecutionErrors{}
	var failedNames []string
	for _, in := range inputs {
		if in.Resources == nil {
			continue
		}
		if err := c.syncOne(ctx, in); err != nil {
			c.log.Error(err, "failed to sync resources", "name", in.Name)
			failedNames = append(failedNames, in.Name)
			var execErrs types.ADCExecutionErrors
			if errors.As(err, &execErrs) {
				failedMap[in.Name] = execErrs
			}
		}
	}

	var err error
	if len(failedNames) > 0 {
		err = fmt.Errorf("failed to sync %d configs: %s",
			len(failedNames),
			strings.Join(failedNames, ", "))
	}
	return failedMap, err
}

// push syncs one config through the ADC server, re-deriving the baseline ADC diffs against
// whenever that baseline cannot be trusted. Beside the error to report it returns the ones
// to report next to it, which a rebuild that failed leaves behind.
//
// The ADC sidecar outlives the controller process, so the baseline it holds for a cacheKey
// may be one an earlier leadership term left behind. It is re-derived from the data plane
// the first time this term syncs the key, before anything can be pushed from it, and only
// a sync ADC accepts settles the question.
//
// Rebuilding on leader acquisition covers where staleness comes from. The safety net covers
// what it cannot foresee -- another writer on this data plane, a desync no leadership change
// explains -- and a conf_version the data plane refuses is the only way any of that shows
// itself. Re-read the data plane and push again.
func (c *Client) push(ctx context.Context, config adctypes.Config, resources *adctypes.Resources, labels map[string]string, resourceTypes []string) ([]types.ADCExecutionError, error) {
	standalone := config.BackendType == backendAPISIXStandalone
	config.BypassCache = standalone && !c.baselineIsCurrent(config.Name)

	err := c.executor.Execute(ctx, config, resources, labels, resourceTypes)

	var alsoReport []types.ADCExecutionError
	if standalone && !config.BypassCache && isConfVersionRejection(err) {
		c.log.Info("data plane rejected a stale conf_version, rebuilding the ADC baseline",
			"config", config.Name, "error", err.Error())
		// Keep the rejection visible even when the sync recovers. The rebuild is not rate
		// limited, so a rejection on every sync -- someone else writing to this data plane --
		// turns every sync into a full fetch and diff, and this counter is what says so.
		pkgmetrics.RecordExecutionError(config.Name, "conf_version_conflict")

		config.BypassCache = true
		retryErr := c.executor.Execute(ctx, config, resources, labels, resourceTypes)

		// Report the rejection as well. On its own a failed rebuild says nothing about what it
		// was rebuilding for, and it is the rejection that names the cause -- an ADC server too
		// old to know bypassCache, say, answers with a schema error that points nowhere near
		// it. Unless the rebuild was rejected the same way, in which case saying it twice only
		// pads the status message.
		var rejected types.ADCExecutionError
		if retryErr != nil && retryErr.Error() != err.Error() && errors.As(err, &rejected) {
			alsoReport = append(alsoReport, rejected)
		}
		err = retryErr
	}

	// Only a sync ADC accepted proves its baseline is now derived from the data plane.
	if err == nil && config.BypassCache {
		c.markBaselineCurrent(config.Name)
	}
	return alsoReport, err
}

func (c *Client) syncOne(ctx context.Context, in SyncInput) error {
	c.log.V(1).Info("syncing resources", "input", in)

	var errs types.ADCExecutionErrors

	config := in.Config
	if config.BackendType == "" {
		config.BackendType = c.defaultMode
	}

	startTime := time.Now()
	resourceType := strings.Join(in.ResourceTypes, ",")
	if resourceType == "" {
		resourceType = "all"
	}

	alsoReport, err := c.push(ctx, config, in.Resources, in.Labels, in.ResourceTypes)
	errs.Errors = append(errs.Errors, alsoReport...)

	duration := time.Since(startTime).Seconds()

	status := adctypes.StatusSuccess
	if err != nil {
		status = "failure"
		c.log.Error(err, "failed to sync with ADC", "config", config)

		var execErr types.ADCExecutionError
		if errors.As(err, &execErr) {
			errs.Errors = append(errs.Errors, execErr)
			pkgmetrics.RecordExecutionError(config.Name, execErr.Name)
		} else {
			errs.Errors = append(errs.Errors, types.ADCExecutionError{
				Name:         config.Name,
				FailedErrors: []types.ADCExecutionServerAddrError{{Err: err.Error()}},
			})
			pkgmetrics.RecordExecutionError(config.Name, "unknown")
		}
	}

	pkgmetrics.RecordSyncDuration(config.Name, resourceType, status, duration)

	if len(errs.Errors) > 0 {
		return errs
	}
	return nil
}
