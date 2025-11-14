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
		executor:         NewHTTPADCExecutor(log, serverURL, timeout),
		ConfigManager:    configManager,
		ADCDebugProvider: common.NewADCDebugProvider(store, configManager),
		log:              logger,
		defaultMode:      defaultMode,
	}, nil
}

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

		err := c.executor.Execute(ctx, config, args)
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
