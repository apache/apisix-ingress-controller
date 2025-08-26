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

	"github.com/api7/gopkg/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"

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

	executor    ADCExecutor
	BackendMode string

	ConfigManager *common.ConfigManager[types.NamespacedNameKind, adctypes.Config]
}

func New(mode string, timeout time.Duration) (*Client, error) {
	serverURL := os.Getenv("ADC_SERVER_URL")
	if serverURL == "" {
		serverURL = defaultHTTPADCExecutorAddr
	}

	log.Infow("using HTTP ADC Executor", zap.String("server_url", serverURL))
	return &Client{
		Store:         cache.NewStore(),
		executor:      NewHTTPADCExecutor(serverURL, timeout),
		BackendMode:   mode,
		ConfigManager: common.NewConfigManager[types.NamespacedNameKind, adctypes.Config](),
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

func (d *Client) Update(ctx context.Context, args Task) error {
	d.mu.Lock()
	deleteConfigs := d.ConfigManager.Update(args.Key, args.Configs)
	for _, config := range deleteConfigs {
		if err := d.Store.Delete(config.Name, args.ResourceTypes, args.Labels); err != nil {
			log.Errorw("failed to delete resources from store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}

	for _, config := range args.Configs {
		if err := d.Insert(config.Name, args.ResourceTypes, args.Resources, args.Labels); err != nil {
			log.Errorw("failed to insert resources into store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}
	d.mu.Unlock()

	d.syncMu.RLock()
	defer d.syncMu.RUnlock()

	if len(deleteConfigs) > 0 {
		err := d.sync(ctx, Task{
			Name:          args.Name,
			Labels:        args.Labels,
			ResourceTypes: args.ResourceTypes,
			Configs:       deleteConfigs,
		})
		if err != nil {
			log.Warnw("failed to sync deleted configs", zap.Error(err))
		}
	}

	return d.sync(ctx, args)
}

func (d *Client) UpdateConfig(ctx context.Context, args Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	deleteConfigs := d.ConfigManager.Update(args.Key, args.Configs)

	for _, config := range deleteConfigs {
		if err := d.Store.Delete(config.Name, args.ResourceTypes, args.Labels); err != nil {
			log.Errorw("failed to delete resources from store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}

	for _, config := range args.Configs {
		if err := d.Insert(config.Name, args.ResourceTypes, args.Resources, args.Labels); err != nil {
			log.Errorw("failed to insert resources into store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}
	return nil
}

func (d *Client) Delete(ctx context.Context, args Task) error {
	d.mu.Lock()
	configs := d.ConfigManager.Get(args.Key)
	d.ConfigManager.Delete(args.Key)

	for _, config := range configs {
		if err := d.Store.Delete(config.Name, args.ResourceTypes, args.Labels); err != nil {
			log.Errorw("failed to delete resources from store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}
	d.mu.Unlock()

	d.syncMu.RLock()
	defer d.syncMu.RUnlock()

	return d.sync(ctx, Task{
		Labels:        args.Labels,
		ResourceTypes: args.ResourceTypes,
		Configs:       configs,
	})
}

func (d *Client) DeleteConfig(ctx context.Context, args Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	configs := d.ConfigManager.Get(args.Key)
	d.ConfigManager.Delete(args.Key)

	for _, config := range configs {
		if err := d.Store.Delete(config.Name, args.ResourceTypes, args.Labels); err != nil {
			log.Errorw("failed to delete resources from store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}

	return nil
}

func (c *Client) Sync(ctx context.Context) (map[string]types.ADCExecutionErrors, error) {
	c.syncMu.Lock()
	defer c.syncMu.Unlock()
	log.Debug("syncing all resources")

	configs := c.ConfigManager.List()

	if len(configs) == 0 {
		log.Warn("no GatewayProxy configs provided")
		return nil, nil
	}

	log.Debugw("syncing resources with multiple configs", zap.Any("configs", configs))

	failedMap := map[string]types.ADCExecutionErrors{}
	var failedConfigs []string
	for _, config := range configs {
		name := config.Name
		resources, err := c.GetResources(name)
		if err != nil {
			log.Errorw("failed to get resources from store", zap.String("name", name), zap.Error(err))
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
			log.Errorw("failed to sync resources", zap.String("name", name), zap.Error(err))
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
	log.Debugw("syncing resources", zap.Any("task", task))

	if len(task.Configs) == 0 {
		log.Warnw("no adc configs provided", zap.Any("task", task))
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
	pkgmetrics.RecordFileIODuration("prepare_sync_file", "success", time.Since(fileIOStart).Seconds())
	defer cleanup()

	args := BuildADCExecuteArgs(syncFilePath, task.Labels, task.ResourceTypes)

	for _, config := range task.Configs {
		// Record sync duration for each config
		startTime := time.Now()
		resourceType := strings.Join(task.ResourceTypes, ",")
		if resourceType == "" {
			resourceType = "all"
		}

		err := c.executor.Execute(ctx, c.BackendMode, config, args)
		duration := time.Since(startTime).Seconds()

		status := "success"
		if err != nil {
			status = "failure"
			log.Errorw("failed to execute adc command", zap.Error(err), zap.Any("config", config))

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

	log.Debugw("generated adc file", zap.String("filename", tmpFile.Name()), zap.String("json", string(data)))

	return tmpFile.Name(), cleanup, nil
}
