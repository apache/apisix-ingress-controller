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
	"time"

	"C"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

// CompareResources use to compare the object IDs in resources and APISIX
// Find out the rest of objects in APISIX
// AND remove them.
func (c *Controller) CompareResources() {
	var (
		routeMapK8S       = new(sync.Map)
		streamRouteMapK8S = new(sync.Map)
		upstreamMapK8S    = new(sync.Map)
		sslMapK8S         = new(sync.Map)
		consumerMapK8S    = new(sync.Map)

		routeMapA6       = new(sync.Map)
		streamRouteMapA6 = new(sync.Map)
		upstreamMapA6    = new(sync.Map)
		sslMapA6         = new(sync.Map)
		consumerMapA6    = new(sync.Map)
	)
	// todo if watchingNamespace == nil
	if c.watchingNamespace == nil {
		opts := v1.ListOptions{}
		// list all apisixroute resources in all namespaces
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
	for ns, _ := range c.watchingNamespace {
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
	}

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
	// 4.remove from APISIX
	c.removeRouteFromA6(routeReult)
	c.removeStreamRouteFromA6(streamRouteReult)
	c.removeSSLFromA6(sslReult)
	c.removeConsumerFromA6(consuemrReult)
	time.Sleep(5 * time.Second)
	c.removeUpstreamFromA6(upstreamReult)
}

// findRedundant find redundant item which in src and do not in dest
func findRedundant(src, dest *sync.Map) *sync.Map {
	result := new(sync.Map)
	src.Range(func(k, v interface{}) bool {
		_, ok := dest.Load(k)
		if !ok {
			result.Store(k, v)
		}
		return true
	})
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

func (c *Controller) listRouteCache(routeMapA6 *sync.Map) {
	routesInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).Route().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, ra := range routesInA6 {
			routeMapA6.Store(ra.ID, ra.ID)
		}
	}
}

func (c *Controller) listStreamRouteCache(streamRouteMapA6 *sync.Map) {
	streamRoutesInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).StreamRoute().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, ra := range streamRoutesInA6 {
			streamRouteMapA6.Store(ra.ID, ra.ID)
		}
	}
}

func (c *Controller) listUpstreamCache(upstreamMapA6 *sync.Map) {
	upstreamsInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).Upstream().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, ra := range upstreamsInA6 {
			upstreamMapA6.Store(ra.ID, ra.ID)
		}
	}
}

func (c *Controller) listSSLCache(sslMapA6 *sync.Map) {
	sslInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).SSL().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, s := range sslInA6 {
			sslMapA6.Store(s.ID, s.ID)
		}
	}
}

func (c *Controller) listConsumerCache(consumerMapA6 *sync.Map) {
	consumerInA6, err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).Consumer().List(context.TODO())
	if err != nil {
		panic(err)
	} else {
		for _, con := range consumerInA6 {
			consumerMapA6.Store(con.Username, con.Username)
		}
	}
}
