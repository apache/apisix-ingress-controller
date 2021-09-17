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
package ingress

import (
	"context"
	"sync"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

// CompareResources use to compare the object IDs in resources and APISIX
// Find out the rest of objects in APISIX
// AND warn them in log.
func (c *Controller) CompareResources() {
	var (
		wg                sync.WaitGroup
		routeMapK8S       = new(sync.Map)
		streamRouteMapK8S = new(sync.Map)
		upstreamMapK8S    = new(sync.Map)
		sslMapK8S         = new(sync.Map)
		consumerMapK8S    = new(sync.Map)

		routeMapA6       = make(map[string]string)
		streamRouteMapA6 = make(map[string]string)
		upstreamMapA6    = make(map[string]string)
		sslMapA6         = make(map[string]string)
		consumerMapA6    = make(map[string]string)
	)
	// watchingNamespace == nil means to monitor all namespaces
	if c.watchingNamespace == nil {
		opts := v1.ListOptions{}
		// list all namespaces
		nsList, err := c.kubeClient.Client.CoreV1().Namespaces().List(context.TODO(), opts)
		if err != nil {
			panic(err)
		} else {
			wns := make(map[string]struct{}, len(nsList.Items))
			for _, v := range nsList.Items {
				wns[v.Name] = struct{}{}
			}
			c.watchingNamespace = wns
		}
	}
	if len(c.watchingNamespace) > 0 {
		wg.Add(len(c.watchingNamespace))
	}
	for ns := range c.watchingNamespace {
		go func() {
			// ApisixRoute
			opts := v1.ListOptions{}
			retRoutes, err := c.kubeClient.APISIXClient.ApisixV2beta1().ApisixRoutes(ns).List(context.TODO(), opts)
			if err != nil {
				panic(err)
			} else {
				for _, r := range retRoutes.Items {
					tc, err := c.translator.TranslateRouteV2beta1NotStrictly(&r)
					if err != nil {
						panic(err)
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
					}
				}
			}
			// todo ApisixUpstream
			// ApisixUpstream should be synced with ApisixRoute resource

			// ApisixSSL
			retSSL, err := c.kubeClient.APISIXClient.ApisixV1().ApisixTlses(ns).List(context.TODO(), opts)
			if err != nil {
				panic(err)
			} else {
				for _, s := range retSSL.Items {
					ssl, err := c.translator.TranslateSSL(&s)
					if err != nil {
						panic(err)
					} else {
						sslMapK8S.Store(ssl.ID, ssl.ID)
					}
				}
			}
			// ApisixConsumer
			retConsumer, err := c.kubeClient.APISIXClient.ApisixV2alpha1().ApisixConsumers(ns).List(context.TODO(), opts)
			if err != nil {
				panic(err)
			} else {
				for _, con := range retConsumer.Items {
					consumer, err := c.translator.TranslateApisixConsumer(&con)
					if err != nil {
						panic(err)
					} else {
						consumerMapK8S.Store(consumer.Username, consumer.Username)
					}
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// 2.get all cache routes
	c.listRouteCache(routeMapA6)
	c.listStreamRouteCache(streamRouteMapA6)
	c.listUpstreamCache(upstreamMapA6)
	c.listSSLCache(sslMapA6)
	c.listConsumerCache(consumerMapA6)
	// 3.compare
	routeReult := findRedundant(routeMapA6, routeMapK8S)
	streamRouteReult := findRedundant(streamRouteMapA6, streamRouteMapK8S)
	upstreamReult := findRedundant(upstreamMapA6, upstreamMapK8S)
	sslReult := findRedundant(sslMapA6, sslMapK8S)
	consuemrReult := findRedundant(consumerMapA6, consumerMapK8S)
	// 4.warn
	warnRedundantResources(routeReult, "route")
	warnRedundantResources(streamRouteReult, "streamRoute")
	warnRedundantResources(upstreamReult, "upstream")
	warnRedundantResources(sslReult, "ssl")
	warnRedundantResources(consuemrReult, "consumer")
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

func (c *Controller) removeConsumerFromA6(consumers *sync.Map) {
	consumers.Range(func(k, v interface{}) bool {
		r := &apisix.Consumer{}
		r.Username = k.(string)
		err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).Consumer().Delete(context.TODO(), r)
		if err != nil {
			panic(err)
		}
		return true
	})
}

func (c *Controller) removeSSLFromA6(sslReult *sync.Map) {
	sslReult.Range(func(k, v interface{}) bool {
		r := &apisix.Ssl{}
		r.ID = k.(string)
		err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).SSL().Delete(context.TODO(), r)
		if err != nil {
			panic(err)
		}
		return true
	})
}

func (c *Controller) removeUpstreamFromA6(upstreamReult *sync.Map) {
	upstreamReult.Range(func(k, v interface{}) bool {
		r := &apisix.Upstream{}
		r.ID = k.(string)
		err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).Upstream().Delete(context.TODO(), r)
		if err != nil {
			panic(err)
		}
		return true
	})
}

func (c *Controller) removeStreamRouteFromA6(streamRouteReult *sync.Map) {
	streamRouteReult.Range(func(k, v interface{}) bool {
		r := &apisix.StreamRoute{}
		r.ID = k.(string)
		err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).StreamRoute().Delete(context.TODO(), r)
		if err != nil {
			panic(err)
		}
		return true
	})
}

func (c *Controller) removeRouteFromA6(routeReult *sync.Map) {
	routeReult.Range(func(k, v interface{}) bool {
		r := &apisix.Route{}
		r.ID = k.(string)
		err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).Route().Delete(context.TODO(), r)
		if err != nil {
			panic(err)
		}
		return true
	})
}

func (c *Controller) listRouteCache(routeMapA6 map[string]string) {
	routesInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).Route().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, ra := range routesInA6 {
			routeMapA6[ra.ID] = ra.ID
		}
	}
}

func (c *Controller) listStreamRouteCache(streamRouteMapA6 map[string]string) {
	streamRoutesInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).StreamRoute().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, ra := range streamRoutesInA6 {
			streamRouteMapA6[ra.ID] = ra.ID
		}
	}
}

func (c *Controller) listUpstreamCache(upstreamMapA6 map[string]string) {
	upstreamsInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).Upstream().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, ra := range upstreamsInA6 {
			upstreamMapA6[ra.ID] = ra.ID
		}
	}
}

func (c *Controller) listSSLCache(sslMapA6 map[string]string) {
	sslInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).SSL().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, s := range sslInA6 {
			sslMapA6[s.ID] = s.ID
		}
	}
}

func (c *Controller) listConsumerCache(consumerMapA6 map[string]string) {
	consumerInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).Consumer().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, con := range consumerInA6 {
			consumerMapA6[con.Username] = con.Username
		}
	}
}
