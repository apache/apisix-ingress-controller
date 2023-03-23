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

package apisix

import (
	"errors"
	"reflect"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var (
	ErrUnknownApisixResourceType = errors.New("unknown apisix resource type")
)

type ResourceTypes interface {
	*v1.Route | *v1.Ssl | *v1.Upstream | *v1.StreamRoute | *v1.GlobalRule | *v1.Consumer | *v1.PluginConfig
}

func skipRequest[T ResourceTypes](cluster *cluster, shouldCompare bool, id string, obj T) (T, bool) {
	if cluster.syncComparison && shouldCompare {
		var (
			// generated object may be different from server response object,
			// so we need another cache to store generated objs
			cachedGeneratedObj interface{}
			err                error
		)

		resourceType := ""
		// type-switch on parametric types is not implemented yet
		switch (interface{})(obj).(type) {
		case *v1.Route:
			cachedGeneratedObj, err = cluster.generatedObjCache.GetRoute(id)
			resourceType = "route"
		case *v1.Ssl:
			cachedGeneratedObj, err = cluster.generatedObjCache.GetSSL(id)
			resourceType = "ssl"
		case *v1.Upstream:
			cachedGeneratedObj, err = cluster.generatedObjCache.GetUpstream(id)
			resourceType = "upstream"
		case *v1.StreamRoute:
			cachedGeneratedObj, err = cluster.generatedObjCache.GetStreamRoute(id)
			resourceType = "stream_route"
		case *v1.GlobalRule:
			cachedGeneratedObj, err = cluster.generatedObjCache.GetGlobalRule(id)
			resourceType = "global_rule"
		case *v1.Consumer:
			cachedGeneratedObj, err = cluster.generatedObjCache.GetConsumer(id)
			resourceType = "consumer"
		case *v1.PluginConfig:
			cachedGeneratedObj, err = cluster.generatedObjCache.GetPluginConfig(id)
			resourceType = "plugin_config"
		//case *v1.PluginMetadata:
		default:
			log.Errorw("resource comparison aborted",
				zap.Error(ErrUnknownApisixResourceType),
				zap.Any("obj", obj),
			)
			return nil, false
		}

		if err == nil && cachedGeneratedObj != nil {
			if reflect.DeepEqual(cachedGeneratedObj, obj) {
				var (
					cachedServerObj interface{}
					err2            error
				)

				switch (interface{})(obj).(type) {
				case *v1.Route:
					cachedServerObj, err2 = cluster.cache.GetRoute(id)
				case *v1.Ssl:
					cachedServerObj, err2 = cluster.cache.GetSSL(id)
				case *v1.Upstream:
					cachedServerObj, err2 = cluster.cache.GetUpstream(id)
				case *v1.StreamRoute:
					cachedServerObj, err2 = cluster.cache.GetStreamRoute(id)
				case *v1.GlobalRule:
					cachedServerObj, err2 = cluster.cache.GetGlobalRule(id)
				case *v1.Consumer:
					cachedServerObj, err2 = cluster.cache.GetConsumer(id)
				case *v1.PluginConfig:
					cachedServerObj, err2 = cluster.cache.GetPluginConfig(id)
				}

				if err2 == nil && cachedServerObj != nil {
					log.Debugw("sync comparison skipped same resource",
						zap.String("reason", "equal"),
						zap.String("resource", resourceType),
						zap.Any("obj", obj),
						zap.Any("cached", cachedGeneratedObj),
					)

					return cachedServerObj.(T), true
				} else {
					log.Debugw("sync comparison continue operation",
						zap.String("reason", "failed to get cached server object"),
						zap.String("resource", resourceType),
						zap.Error(err2),
						zap.String("id", id),
					)

					return nil, false
				}
			} else {
				log.Debugw("sync comparison continue operation",
					zap.String("reason", "not equal"),
					zap.String("resource", resourceType),
					zap.Any("obj", obj),
					zap.Any("cached", cachedGeneratedObj),
				)
			}
		} else if err == cache.ErrNotFound {
			log.Debugw("sync comparison continue operation",
				zap.String("reason", "not in cache"),
				zap.String("resource", resourceType),
				zap.String("id", id),
				zap.Any("obj", obj),
				zap.Any("cached", cachedGeneratedObj),
			)
		} else {
			log.Debugw("sync comparison continue operation",
				zap.Error(err),
				zap.String("reason", "failed to get cached generated object"),
				zap.String("resource", resourceType),
				zap.String("id", id),
				zap.Any("obj", obj),
				zap.Any("cached", cachedGeneratedObj),
			)
		}
	}

	return nil, false
}
