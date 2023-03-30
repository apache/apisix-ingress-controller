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
	"context"
	"encoding/json"
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

func skipRequest[T ResourceTypes](cluster *cluster, shouldCompare bool, url, id string, obj T) (T, bool) {
	if cluster.syncComparison && shouldCompare {
		var generatedObj T
		resourceType := ""

		// GlobalRule and Consumer has Plugins field which type will mismatch in DeepEqual
		switch (interface{})(generatedObj).(type) {
		case *v1.GlobalRule:
			generatedObj = obj
			resourceType = "global_rule"
		case *v1.Consumer:
			generatedObj = obj
			resourceType = "consumer"
		}
		if generatedObj == nil {
			generatedObj = obj
		} else {
			j, err := json.Marshal(generatedObj)
			if err != nil {
				log.Debugw("sync comparison continue operation",
					zap.String("reason", "failed to marshal object"),
					zap.Error(err),
					zap.Any("resource", resourceType),
					zap.Any("obj", generatedObj),
				)
				return nil, false
			}
			err = json.Unmarshal(j, &generatedObj)
			if err != nil {
				log.Debugw("sync comparison continue operation",
					zap.String("reason", "failed to unmarshal object"),
					zap.Error(err),
					zap.Any("resource", resourceType),
					zap.Any("json", j),
				)
				return nil, false
			}
		}

		var (
			// generated object may be different from server response object,
			// so we need another cache to store generated objs
			cachedGeneratedObj interface{}
			err                error
		)

		// type-switch on parametric types is not implemented yet
		switch (interface{})(generatedObj).(type) {
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
				zap.Any("obj", generatedObj),
			)
			return nil, false
		}

		if err == nil && cachedGeneratedObj != nil {
			if reflect.DeepEqual(cachedGeneratedObj, generatedObj) {
				var (
					expectedServerObj interface{}
				)

				switch (interface{})(generatedObj).(type) {
				case *v1.Route:
					expectedServerObj, err = cluster.cache.GetRoute(id)
				case *v1.Ssl:
					expectedServerObj, err = cluster.cache.GetSSL(id)
					if err == nil && expectedServerObj != nil {
						expectedServerObj.(*v1.Ssl).Key = ""
					}
				case *v1.Upstream:
					expectedServerObj, err = cluster.cache.GetUpstream(id)
				case *v1.StreamRoute:
					expectedServerObj, err = cluster.cache.GetStreamRoute(id)
				case *v1.GlobalRule:
					expectedServerObj, err = cluster.cache.GetGlobalRule(id)
				case *v1.Consumer:
					expectedServerObj, err = cluster.cache.GetConsumer(id)
				case *v1.PluginConfig:
					expectedServerObj, err = cluster.cache.GetPluginConfig(id)
				}

				if err == nil && expectedServerObj != nil {
					// Now we have the expected server obj, compare to actual object in APISIX

					var (
						serverObj interface{}
					)
					switch (interface{})(generatedObj).(type) {
					case *v1.Route:
						serverObj, err = cluster.GetRoute(context.Background(), url, id)
					case *v1.Ssl:
						serverObj, err = cluster.GetSSL(context.Background(), url, id)
					case *v1.Upstream:
						serverObj, err = cluster.GetUpstream(context.Background(), url, id)
					case *v1.StreamRoute:
						serverObj, err = cluster.GetStreamRoute(context.Background(), url, id)
					case *v1.GlobalRule:
						serverObj, err = cluster.GetGlobalRule(context.Background(), url, id)
					case *v1.Consumer:
						serverObj, err = cluster.GetConsumer(context.Background(), url, id)
					case *v1.PluginConfig:
						serverObj, err = cluster.GetPluginConfig(context.Background(), url, id)
					}
					if err == nil && serverObj != nil {
						if reflect.DeepEqual(expectedServerObj, serverObj) {
							log.Debugw("sync comparison skipped same resource",
								zap.String("reason", "equal"),
								zap.String("resource", resourceType),
								zap.Any("obj", generatedObj),
								zap.Any("cached", cachedGeneratedObj),
							)

							return expectedServerObj.(T), true
						}

						log.Debugw("sync comparison continue operation",
							zap.String("reason", "cached server object doesn't match APISIX object"),
							zap.String("resource", resourceType),
							zap.Error(err),
							zap.String("id", id),
							zap.Any("cached_obj", expectedServerObj),
							zap.Any("server_obj", serverObj),
						)
						return nil, false
					}

					log.Debugw("sync comparison continue operation",
						zap.String("reason", "failed to get object from APISIX"),
						zap.String("resource", resourceType),
						zap.Error(err),
						zap.String("id", id),
					)

					return nil, false
				}

				log.Debugw("sync comparison continue operation",
					zap.String("reason", "failed to get cached server object"),
					zap.String("resource", resourceType),
					zap.Error(err),
					zap.String("id", id),
				)

				return nil, false
			}

			log.Debugw("sync comparison continue operation",
				zap.String("reason", "controller generated object doesn't match"),
				zap.String("resource", resourceType),
				zap.Any("obj", generatedObj),
				zap.Any("cached", cachedGeneratedObj),
			)

			return nil, false
		} else if err == cache.ErrNotFound {
			log.Debugw("sync comparison continue operation",
				zap.String("reason", "not in cache"),
				zap.String("resource", resourceType),
				zap.String("id", id),
				zap.Any("obj", generatedObj),
				zap.Any("cached", cachedGeneratedObj),
			)

			return nil, false
		} else {
			log.Debugw("sync comparison continue operation",
				zap.Error(err),
				zap.String("reason", "failed to get cached generated object"),
				zap.String("resource", resourceType),
				zap.String("id", id),
				zap.Any("obj", generatedObj),
				zap.Any("cached", cachedGeneratedObj),
			)

			return nil, false
		}
	}

	return nil, false
}
