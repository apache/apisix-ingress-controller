// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package apisix

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

// Init used to compare the object IDs in resources and APISIX
// Find out the rest of objects in APISIX
// AND warn them in log.
// This func is NOT concurrency safe.
// cc https://github.com/apache/apisix-ingress-controller/pull/742#discussion_r757197791
func (p *apisixProvider) Init(ctx context.Context) error {
	var (
		wg                 sync.WaitGroup
		routeMapK8S        = new(sync.Map)
		streamRouteMapK8S  = new(sync.Map)
		upstreamMapK8S     = new(sync.Map)
		sslMapK8S          = new(sync.Map)
		consumerMapK8S     = new(sync.Map)
		pluginConfigMapK8S = new(sync.Map)

		routeMapA6        = make(map[string]string)
		streamRouteMapA6  = make(map[string]string)
		upstreamMapA6     = make(map[string]string)
		sslMapA6          = make(map[string]string)
		consumerMapA6     = make(map[string]string)
		pluginConfigMapA6 = make(map[string]string)
	)

	namespaces := p.namespaceProvider.WatchingNamespaces()

	for _, key := range namespaces {
		log.Debugf("start to watch namespace: %s", key)
		wg.Add(1)
		go func(ns string) {
			defer wg.Done()
			// ApisixRoute
			switch p.common.Config.Kubernetes.APIVersion {
			case config.ApisixV2:
				retRoutes, err := p.common.ApisixRouteLister.V2Lister().List(labels.Everything())
				if err != nil {
					log.Error(err.Error())
					ctx.Done()
				} else {
					for _, r := range retRoutes {
						tc, err := p.apisixTranslator.GenerateRouteV2DeleteMark(r)
						if err != nil {
							log.Error(err.Error())
							ctx.Done()
						} else {
							// routes
							for _, route := range tc.Routes {
								routeMapK8S.Store(route.ID, route.ID)
							}
							// streamRoutes
							for _, stRoute := range tc.StreamRoutes {
								streamRouteMapK8S.Store(stRoute.ID, stRoute.ID)
							}
							// upstreams
							for _, upstream := range tc.Upstreams {
								upstreamMapK8S.Store(upstream.ID, upstream.ID)
							}
							// ssl
							for _, ssl := range tc.SSL {
								sslMapK8S.Store(ssl.ID, ssl.ID)
							}
							// pluginConfigs
							for _, pluginConfig := range tc.PluginConfigs {
								pluginConfigMapK8S.Store(pluginConfig.ID, pluginConfig.ID)
							}
						}
					}
				}
			default:
				log.Errorw("failed to sync ApisixRoute, unexpected version",
					zap.String("version", p.common.Config.Kubernetes.APIVersion),
				)
			}
			// todo ApisixUpstream and ApisixPluginConfig
			// ApisixUpstream and ApisixPluginConfig should be synced with ApisixRoute resource

			switch p.common.Config.Kubernetes.APIVersion {
			case config.ApisixV2:
				// ApisixConsumer
				retConsumer, err := p.common.ApisixFactory.Apisix().V2().ApisixConsumers().Lister().ApisixConsumers(ns).List(labels.Everything())
				if err != nil {
					log.Error(err.Error())
					ctx.Done()
				} else {
					for _, con := range retConsumer {
						consumer, err := p.apisixTranslator.TranslateApisixConsumerV2(con)
						if err != nil {
							log.Error(err.Error())
							ctx.Done()
						} else {
							consumerMapK8S.Store(consumer.Username, consumer.Username)
						}
					}
				}
				// ApisixTls
				retSSL, err := p.common.ApisixFactory.Apisix().V2().ApisixTlses().Lister().ApisixTlses(ns).List(labels.Everything())
				if err != nil {
					log.Error(err.Error())
					ctx.Done()
				} else {
					for _, s := range retSSL {
						ssl, err := p.apisixTranslator.TranslateSSLV2(s)
						if err != nil {
							log.Error(err.Error())
							ctx.Done()
						} else {
							sslMapK8S.Store(ssl.ID, ssl.ID)
						}
					}
				}
			default:
				log.Errorw("failed to sync ApisixConsumer, unexpected version",
					zap.String("version", p.common.Config.Kubernetes.APIVersion),
				)
			}
		}(key)
	}
	wg.Wait()

	// 2.get all cache routes
	if err := p.listRouteCache(ctx, routeMapA6); err != nil {
		return err
	}
	if err := p.listStreamRouteCache(ctx, streamRouteMapA6); err != nil {
		return err
	}
	if err := p.listUpstreamCache(ctx, upstreamMapA6); err != nil {
		return err
	}
	if err := p.listSSLCache(ctx, sslMapA6); err != nil {
		return err
	}
	if err := p.listConsumerCache(ctx, consumerMapA6); err != nil {
		return err
	}
	if err := p.listPluginConfigCache(ctx, pluginConfigMapA6); err != nil {
		return err
	}
	// 3.compare
	routeResult := findRedundant(routeMapA6, routeMapK8S)
	streamRouteResult := findRedundant(streamRouteMapA6, streamRouteMapK8S)
	upstreamResult := findRedundant(upstreamMapA6, upstreamMapK8S)
	sslResult := findRedundant(sslMapA6, sslMapK8S)
	consumerResult := findRedundant(consumerMapA6, consumerMapK8S)
	pluginConfigResult := findRedundant(pluginConfigMapA6, pluginConfigMapK8S)
	// 4.warn
	warnRedundantResources(routeResult, "route")
	warnRedundantResources(streamRouteResult, "streamRoute")
	warnRedundantResources(upstreamResult, "upstream")
	warnRedundantResources(sslResult, "ssl")
	warnRedundantResources(consumerResult, "consumer")
	warnRedundantResources(pluginConfigResult, "pluginConfig")

	return nil
}

// log warn
func warnRedundantResources(resources map[string]string, t string) {
	for k := range resources {
		log.Warnf("%s: %s in APISIX but do not in declare yaml", t, k)
	}
}

// findRedundant find redundant item which in src and do not in dest
func findRedundant(src map[string]string, dest *sync.Map) map[string]string {
	result := make(map[string]string)
	for k, v := range src {
		_, ok := dest.Load(k)
		if !ok {
			result[k] = v
		}
	}
	return result
}

func (p *apisixProvider) listRouteCache(ctx context.Context, routeMapA6 map[string]string) error {
	routesInA6, err := p.common.APISIX.Cluster(p.common.Config.APISIX.DefaultClusterName).Route().List(ctx)
	if err != nil {
		return err
	} else {
		for _, ra := range routesInA6 {
			routeMapA6[ra.ID] = ra.ID
		}
	}
	return nil
}

func (p *apisixProvider) listStreamRouteCache(ctx context.Context, streamRouteMapA6 map[string]string) error {
	streamRoutesInA6, err := p.common.APISIX.Cluster(p.common.Config.APISIX.DefaultClusterName).StreamRoute().List(ctx)
	if err != nil {
		return err
	} else {
		for _, ra := range streamRoutesInA6 {
			streamRouteMapA6[ra.ID] = ra.ID
		}
	}
	return nil
}

func (p *apisixProvider) listUpstreamCache(ctx context.Context, upstreamMapA6 map[string]string) error {
	upstreamsInA6, err := p.common.APISIX.Cluster(p.common.Config.APISIX.DefaultClusterName).Upstream().List(ctx)
	if err != nil {
		return err
	} else {
		for _, ra := range upstreamsInA6 {
			upstreamMapA6[ra.ID] = ra.ID
		}
	}
	return nil
}

func (p *apisixProvider) listSSLCache(ctx context.Context, sslMapA6 map[string]string) error {
	sslInA6, err := p.common.APISIX.Cluster(p.common.Config.APISIX.DefaultClusterName).SSL().List(ctx)
	if err != nil {
		return err
	} else {
		for _, s := range sslInA6 {
			sslMapA6[s.ID] = s.ID
		}
	}
	return nil
}

func (p *apisixProvider) listConsumerCache(ctx context.Context, consumerMapA6 map[string]string) error {
	consumerInA6, err := p.common.APISIX.Cluster(p.common.Config.APISIX.DefaultClusterName).Consumer().List(ctx)
	if err != nil {
		return err
	} else {
		for _, con := range consumerInA6 {
			consumerMapA6[con.Username] = con.Username
		}
	}
	return nil
}

func (p *apisixProvider) listPluginConfigCache(ctx context.Context, pluginConfigMapA6 map[string]string) error {
	pluginConfigInA6, err := p.common.APISIX.Cluster(p.common.Config.APISIX.DefaultClusterName).PluginConfig().List(ctx)
	if err != nil {
		return err
	} else {
		for _, ra := range pluginConfigInA6 {
			pluginConfigMapA6[ra.ID] = ra.ID
		}
	}
	return nil
}
