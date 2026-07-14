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

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	"github.com/apache/apisix-ingress-controller/internal/provider/common"
	"github.com/apache/apisix-ingress-controller/internal/types"
	pkgmetrics "github.com/apache/apisix-ingress-controller/pkg/metrics"
)

type Client struct {
	syncMu sync.RWMutex
	mu     sync.Mutex
	*cache.Store

	executor ADCExecutor

	ConfigManager    *common.ConfigManager[types.NamespacedNameKind, adctypes.Config]
	ADCDebugProvider *common.ADCDebugProvider

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
	store := cache.NewStore(log)
	configManager := common.NewConfigManager[types.NamespacedNameKind, adctypes.Config]()

	logger := log.WithName("client")
	logger.Info("ADC client initialized")

	return &Client{
		Store:            store,
		rebuiltBaselines: make(map[string]struct{}),
		executor:         NewHTTPADCExecutor(log, serverURL, timeout),
		ConfigManager:    configManager,
		ADCDebugProvider: common.NewADCDebugProvider(store, configManager),
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

type Task struct {
	Key           types.NamespacedNameKind
	Name          string
	Labels        map[string]string
	Configs       map[types.NamespacedNameKind]adctypes.Config
	ResourceTypes []string
	Resources     *adctypes.Resources
}

type StoreDelta struct {
	Deleted map[types.NamespacedNameKind]adctypes.Config
	Applied map[types.NamespacedNameKind]adctypes.Config
}

func (c *Client) applyStoreChanges(args Task, isDelete bool) (StoreDelta, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var delta StoreDelta

	if isDelete {
		delta.Deleted = c.ConfigManager.Get(args.Key)
		c.ConfigManager.Delete(args.Key)
	} else {
		deleted := c.ConfigManager.Update(args.Key, args.Configs)
		delta.Deleted = deleted
		delta.Applied = args.Configs
	}

	for _, cfg := range delta.Deleted {
		if err := c.Store.Delete(cfg.Name, args.ResourceTypes, args.Labels); err != nil {
			c.log.Error(err, "store delete failed", "cfg", cfg, "args", args)
			return StoreDelta{}, errors.Wrap(err, fmt.Sprintf("store delete failed for config %s", cfg.Name))
		}
	}

	for _, cfg := range delta.Applied {
		if err := c.Insert(cfg.Name, args.ResourceTypes, args.Resources, args.Labels); err != nil {
			c.log.Error(err, "store insert failed", "cfg", cfg, "args", args)
			return StoreDelta{}, errors.Wrap(err, fmt.Sprintf("store insert failed for config %s", cfg.Name))
		}
	}

	return delta, nil
}

func (c *Client) applySync(ctx context.Context, args Task, delta StoreDelta) error {
	c.syncMu.RLock()
	defer c.syncMu.RUnlock()

	if len(delta.Deleted) > 0 {
		if err := c.sync(ctx, Task{
			Name:          args.Name,
			Labels:        args.Labels,
			ResourceTypes: args.ResourceTypes,
			Configs:       delta.Deleted,
		}); err != nil {
			c.log.Error(err, "failed to sync deleted configs", "args", args, "delta", delta)
		}
	}

	if len(delta.Applied) > 0 {
		return c.sync(ctx, Task{
			Name:          args.Name,
			Labels:        args.Labels,
			ResourceTypes: args.ResourceTypes,
			Configs:       delta.Applied,
			Resources:     args.Resources,
		})
	}
	return nil
}

func (c *Client) Update(ctx context.Context, args Task) error {
	delta, err := c.applyStoreChanges(args, false)
	if err != nil {
		return err
	}
	return c.applySync(ctx, args, delta)
}

func (c *Client) UpdateConfig(ctx context.Context, args Task) error {
	_, err := c.applyStoreChanges(args, false)
	return err
}

func (c *Client) Delete(ctx context.Context, args Task) error {
	delta, err := c.applyStoreChanges(args, true)
	if err != nil {
		return err
	}
	return c.applySync(ctx, args, delta)
}

func (c *Client) DeleteConfig(ctx context.Context, args Task) error {
	_, err := c.applyStoreChanges(args, true)
	return err
}

func (c *Client) Validate(ctx context.Context, task Task) error {
	if len(task.Configs) == 0 || task.Resources == nil {
		return nil
	}

	fileIOStart := time.Now()
	syncFilePath, cleanup, err := prepareSyncFile(task.Resources)
	if err != nil {
		pkgmetrics.RecordFileIODuration("prepare_sync_file", "failure", time.Since(fileIOStart).Seconds())
		return err
	}
	pkgmetrics.RecordFileIODuration("prepare_sync_file", adctypes.StatusSuccess, time.Since(fileIOStart).Seconds())
	defer cleanup()

	args2 := BuildADCExecuteArgs(syncFilePath, task.Labels, task.ResourceTypes)

	var errs types.ADCValidationErrors
	for _, config := range task.Configs {
		if config.BackendType == "" {
			config.BackendType = c.defaultMode
		}
		if err := c.executor.Validate(ctx, config, args2); err != nil {
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

func (c *Client) Sync(ctx context.Context) (map[string]types.ADCExecutionErrors, error) {
	c.syncMu.Lock()
	defer c.syncMu.Unlock()
	c.log.Info("syncing all resources")

	configs := c.ConfigManager.List()

	if len(configs) == 0 {
		c.log.Info("no GatewayProxy configs provided")
		return nil, nil
	}

	c.log.V(1).Info("syncing resources with multiple configs", "configs", configs)

	failedMap := map[string]types.ADCExecutionErrors{}
	var failedConfigs []string
	for _, config := range configs {
		name := config.Name
		resources, err := c.GetResources(name)
		if err != nil {
			c.log.Error(err, "failed to get resources from store", "name", name)
			failedConfigs = append(failedConfigs, name)
			continue
		}
		if resources == nil {
			continue
		}
		c.log.Info("syncing resources for config", "service_number", len(resources.Services))

		if err := c.sync(ctx, Task{
			Name: name + "-sync",
			Configs: map[types.NamespacedNameKind]adctypes.Config{
				{}: config,
			},
			Resources: resources,
		}); err != nil {
			c.log.Error(err, "failed to sync resources", "name", name)
			failedConfigs = append(failedConfigs, name)
			var execErrs types.ADCExecutionErrors
			if errors.As(err, &execErrs) {
				failedMap[name] = execErrs
			}
		}
	}

	var err error
	if len(failedConfigs) > 0 {
		err = fmt.Errorf("failed to sync %d configs: %s",
			len(failedConfigs),
			strings.Join(failedConfigs, ", "))
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
func (c *Client) push(ctx context.Context, config adctypes.Config, args []string) ([]types.ADCExecutionError, error) {
	standalone := config.BackendType == backendAPISIXStandalone
	config.BypassCache = standalone && !c.baselineIsCurrent(config.Name)

	err := c.executor.Execute(ctx, config, args)

	var alsoReport []types.ADCExecutionError
	if standalone && !config.BypassCache && isConfVersionRejection(err) {
		c.log.Info("data plane rejected a stale conf_version, rebuilding the ADC baseline",
			"config", config.Name, "error", err.Error())
		// Keep the rejection visible even when the sync recovers. The rebuild is not rate
		// limited, so a rejection on every sync -- someone else writing to this data plane --
		// turns every sync into a full fetch and diff, and this counter is what says so.
		pkgmetrics.RecordExecutionError(config.Name, "conf_version_conflict")

		config.BypassCache = true
		retryErr := c.executor.Execute(ctx, config, args)

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

func (c *Client) sync(ctx context.Context, task Task) error {
	c.log.V(1).Info("syncing resources", "task", task)

	if len(task.Configs) == 0 {
		c.log.Info("no adc configs provided")
		return nil
	}

	var errs types.ADCExecutionErrors

	// Record file I/O duration
	fileIOStart := time.Now()
	// every task resources is the same, so we can use the first config to prepare the sync file
	syncFilePath, cleanup, err := prepareSyncFile(task.Resources)
	if err != nil {
		pkgmetrics.RecordFileIODuration("prepare_sync_file", "failure", time.Since(fileIOStart).Seconds())
		return err
	}
	pkgmetrics.RecordFileIODuration("prepare_sync_file", adctypes.StatusSuccess, time.Since(fileIOStart).Seconds())
	defer cleanup()
	c.log.V(1).Info("prepared sync file", "path", syncFilePath)

	args := BuildADCExecuteArgs(syncFilePath, task.Labels, task.ResourceTypes)

	for _, config := range task.Configs {
		// Record sync duration for each config
		startTime := time.Now()
		resourceType := strings.Join(task.ResourceTypes, ",")
		if resourceType == "" {
			resourceType = "all"
		}
		if config.BackendType == "" {
			config.BackendType = c.defaultMode
		}

		alsoReport, err := c.push(ctx, config, args)
		errs.Errors = append(errs.Errors, alsoReport...)

		duration := time.Since(startTime).Seconds()

		status := adctypes.StatusSuccess
		if err != nil {
			status = "failure"
			c.log.Error(err, "failed to execute adc command", "config", config)

			var execErr types.ADCExecutionError
			if errors.As(err, &execErr) {
				errs.Errors = append(errs.Errors, execErr)
				pkgmetrics.RecordExecutionError(config.Name, execErr.Name)
			} else {
				pkgmetrics.RecordExecutionError(config.Name, "unknown")
			}
		}

		// Record metrics
		pkgmetrics.RecordSyncDuration(config.Name, resourceType, status, duration)
	}

	if len(errs.Errors) > 0 {
		return errs
	}
	return nil
}

func prepareSyncFile(resources any) (string, func(), error) {
	data, err := json.Marshal(resources)
	if err != nil {
		return "", nil, err
	}

	tmpFile, err := os.CreateTemp("", "adc-task-*.json")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}
	if _, err := tmpFile.Write(data); err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpFile.Name(), cleanup, nil
}
