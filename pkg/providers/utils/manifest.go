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
package utils

import (
	"context"
	"reflect"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func DiffSSL(olds, news []*apisixv1.Ssl) (added, updated, deleted []*apisixv1.Ssl) {
	if olds == nil {
		return news, nil, nil
	}
	if news == nil {
		return nil, nil, olds
	}

	oldMap := make(map[string]*apisixv1.Ssl, len(olds))
	newMap := make(map[string]*apisixv1.Ssl, len(news))
	for _, ssl := range olds {
		oldMap[ssl.ID] = ssl
	}
	for _, ssl := range news {
		newMap[ssl.ID] = ssl
	}

	for _, ssl := range news {
		if or, ok := oldMap[ssl.ID]; !ok {
			added = append(added, ssl)
		} else if !reflect.DeepEqual(or, ssl) {
			updated = append(updated, ssl)
		}
	}
	for _, ssl := range olds {
		if _, ok := newMap[ssl.ID]; !ok {
			deleted = append(deleted, ssl)
		}
	}
	return
}

func DiffRoutes(olds, news []*apisixv1.Route) (added, updated, deleted []*apisixv1.Route) {
	if olds == nil {
		return news, nil, nil
	}
	if news == nil {
		return nil, nil, olds
	}

	oldMap := make(map[string]*apisixv1.Route, len(olds))
	newMap := make(map[string]*apisixv1.Route, len(news))
	for _, r := range olds {
		oldMap[r.ID] = r
	}
	for _, r := range news {
		newMap[r.ID] = r
	}

	for _, r := range news {
		if or, ok := oldMap[r.ID]; !ok {
			added = append(added, r)
		} else if !reflect.DeepEqual(or, r) {
			updated = append(updated, r)
		}
	}
	for _, r := range olds {
		if _, ok := newMap[r.ID]; !ok {
			deleted = append(deleted, r)
		}
	}
	return
}

func DiffUpstreams(olds, news []*apisixv1.Upstream) (added, updated, deleted []*apisixv1.Upstream) {
	oldMap := make(map[string]*apisixv1.Upstream, len(olds))
	newMap := make(map[string]*apisixv1.Upstream, len(news))
	for _, u := range olds {
		oldMap[u.ID] = u
	}
	for _, u := range news {
		newMap[u.ID] = u
	}

	for _, u := range news {
		if ou, ok := oldMap[u.ID]; !ok {
			added = append(added, u)
		} else if !reflect.DeepEqual(ou, u) {
			updated = append(updated, u)
		}
	}
	for _, u := range olds {
		if _, ok := newMap[u.ID]; !ok {
			deleted = append(deleted, u)
		}
	}
	return
}

func DiffStreamRoutes(olds, news []*apisixv1.StreamRoute) (added, updated, deleted []*apisixv1.StreamRoute) {
	oldMap := make(map[string]*apisixv1.StreamRoute, len(olds))
	newMap := make(map[string]*apisixv1.StreamRoute, len(news))
	for _, sr := range olds {
		oldMap[sr.ID] = sr
	}
	for _, sr := range news {
		newMap[sr.ID] = sr
	}

	for _, sr := range news {
		if ou, ok := oldMap[sr.ID]; !ok {
			added = append(added, sr)
		} else if !reflect.DeepEqual(ou, sr) {
			updated = append(updated, sr)
		}
	}
	for _, sr := range olds {
		if _, ok := newMap[sr.ID]; !ok {
			deleted = append(deleted, sr)
		}
	}
	return
}

func DiffPluginConfigs(olds, news []*apisixv1.PluginConfig) (added, updated, deleted []*apisixv1.PluginConfig) {
	oldMap := make(map[string]*apisixv1.PluginConfig, len(olds))
	newMap := make(map[string]*apisixv1.PluginConfig, len(news))
	for _, sr := range olds {
		oldMap[sr.ID] = sr
	}
	for _, sr := range news {
		newMap[sr.ID] = sr
	}

	for _, sr := range news {
		if ou, ok := oldMap[sr.ID]; !ok {
			added = append(added, sr)
		} else if !reflect.DeepEqual(ou, sr) {
			updated = append(updated, sr)
		}
	}
	for _, sr := range olds {
		if _, ok := newMap[sr.ID]; !ok {
			deleted = append(deleted, sr)
		}
	}
	return
}

func DiffPluginMetadatas(olds, news []*apisixv1.PluginMetadata) (added, updated, deleted []*apisixv1.PluginMetadata) {
	oldMap := make(map[string]*apisixv1.PluginMetadata, len(olds))
	newMap := make(map[string]*apisixv1.PluginMetadata, len(news))
	for _, pm := range olds {
		oldMap[pm.Name] = pm
	}
	for _, pm := range news {
		newMap[pm.Name] = pm
	}

	for _, pm := range news {
		if ou, ok := oldMap[pm.Name]; !ok {
			added = append(added, pm)
		} else if !reflect.DeepEqual(ou, pm) {
			updated = append(updated, pm)
		}
	}
	for _, pm := range olds {
		if _, ok := newMap[pm.Name]; !ok {
			deleted = append(deleted, pm)
		}
	}
	return
}

func DiffGlobalRules(olds, news []*apisixv1.GlobalRule) (added, updated, deleted []*apisixv1.GlobalRule) {
	oldMap := make(map[string]*apisixv1.GlobalRule, len(olds))
	newMap := make(map[string]*apisixv1.GlobalRule, len(news))
	for _, gr := range olds {
		oldMap[gr.ID] = gr
	}
	for _, gr := range news {
		newMap[gr.ID] = gr
	}

	for _, gr := range news {
		if ou, ok := oldMap[gr.ID]; !ok {
			added = append(added, gr)
		} else if !reflect.DeepEqual(ou, gr) {
			updated = append(updated, gr)
		}
	}
	for _, gr := range olds {
		if _, ok := newMap[gr.ID]; !ok {
			deleted = append(deleted, gr)
		}
	}
	return
}

type Manifest struct {
	Routes          []*apisixv1.Route
	Upstreams       []*apisixv1.Upstream
	StreamRoutes    []*apisixv1.StreamRoute
	SSLs            []*apisixv1.Ssl
	PluginConfigs   []*apisixv1.PluginConfig
	PluginMetadatas []*apisixv1.PluginMetadata
	GlobalRules     []*apisixv1.GlobalRule
}

func (m *Manifest) Diff(om *Manifest) (added, updated, deleted *Manifest) {
	sa, su, sd := DiffSSL(om.SSLs, m.SSLs)
	ar, ur, dr := DiffRoutes(om.Routes, m.Routes)
	au, uu, du := DiffUpstreams(om.Upstreams, m.Upstreams)
	asr, usr, dsr := DiffStreamRoutes(om.StreamRoutes, m.StreamRoutes)
	apc, upc, dpc := DiffPluginConfigs(om.PluginConfigs, m.PluginConfigs)
	apm, upm, dpm := DiffPluginMetadatas(om.PluginMetadatas, m.PluginMetadatas)
	agr, ugr, dgr := DiffGlobalRules(om.GlobalRules, m.GlobalRules)

	added = &Manifest{
		Routes:          ar,
		Upstreams:       au,
		StreamRoutes:    asr,
		SSLs:            sa,
		PluginConfigs:   apc,
		PluginMetadatas: apm,
		GlobalRules:     agr,
	}
	updated = &Manifest{
		Routes:          ur,
		Upstreams:       uu,
		StreamRoutes:    usr,
		SSLs:            su,
		PluginConfigs:   upc,
		PluginMetadatas: upm,
		GlobalRules:     ugr,
	}
	deleted = &Manifest{
		Routes:          dr,
		Upstreams:       du,
		StreamRoutes:    dsr,
		SSLs:            sd,
		PluginConfigs:   dpc,
		PluginMetadatas: dpm,
		GlobalRules:     dgr,
	}
	return
}

// Due to dependency, delete priority should be last
// shouldCompare only affects Create event since periodic synchronization is considered as Add event
func SyncManifests(ctx context.Context, apisix apisix.APISIX, clusterName string, added, updated, deleted *Manifest, shouldCompare bool) error {
	var merr *multierror.Error

	if added != nil {
		// Should create upstreams firstly due to the dependencies.
		for _, ssl := range added.SSLs {
			if _, err := apisix.Cluster(clusterName).SSL().Create(ctx, ssl, shouldCompare); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, u := range added.Upstreams {
			if _, err := apisix.Cluster(clusterName).Upstream().Create(ctx, u, shouldCompare); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, pc := range added.PluginConfigs {
			if _, err := apisix.Cluster(clusterName).PluginConfig().Create(ctx, pc, shouldCompare); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, r := range added.Routes {
			if _, err := apisix.Cluster(clusterName).Route().Create(ctx, r, shouldCompare); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, sr := range added.StreamRoutes {
			if _, err := apisix.Cluster(clusterName).StreamRoute().Create(ctx, sr, shouldCompare); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, pm := range added.PluginMetadatas {
			if _, err := apisix.Cluster(clusterName).PluginMetadata().Create(ctx, pm, shouldCompare); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, gr := range added.GlobalRules {
			if _, err := apisix.Cluster(clusterName).GlobalRule().Create(ctx, gr, shouldCompare); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
	}
	if updated != nil {
		for _, ssl := range updated.SSLs {
			if _, err := apisix.Cluster(clusterName).SSL().Update(ctx, ssl, false); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, r := range updated.Upstreams {
			if _, err := apisix.Cluster(clusterName).Upstream().Update(ctx, r, false); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, pc := range updated.PluginConfigs {
			if _, err := apisix.Cluster(clusterName).PluginConfig().Update(ctx, pc, false); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, r := range updated.Routes {
			if _, err := apisix.Cluster(clusterName).Route().Update(ctx, r, false); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, sr := range updated.StreamRoutes {
			if _, err := apisix.Cluster(clusterName).StreamRoute().Update(ctx, sr, false); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, pm := range updated.PluginMetadatas {
			if _, err := apisix.Cluster(clusterName).PluginMetadata().Update(ctx, pm, false); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, gr := range updated.GlobalRules {
			if _, err := apisix.Cluster(clusterName).GlobalRule().Update(ctx, gr, false); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
	}
	if deleted != nil {
		for _, ssl := range deleted.SSLs {
			if err := apisix.Cluster(clusterName).SSL().Delete(ctx, ssl); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, r := range deleted.Routes {
			if err := apisix.Cluster(clusterName).Route().Delete(ctx, r); err != nil {
				log.Warnw("failed to delete route, this may affect upstream deletions",
					zap.Error(err),
					zap.Any("route", r),
				)
				merr = multierror.Append(merr, err)
			}
		}
		for _, sr := range deleted.StreamRoutes {
			if err := apisix.Cluster(clusterName).StreamRoute().Delete(ctx, sr); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, u := range deleted.Upstreams {
			if err := apisix.Cluster(clusterName).Upstream().Delete(ctx, u); err != nil {
				// Upstream might be referenced by other routes.
				if err != cache.ErrStillInUse {
					merr = multierror.Append(merr, err)
				} else {
					log.Infow("upstream was referenced by other routes",
						zap.String("upstream_id", u.ID),
						zap.String("upstream_name", u.Name),
					)

					if log.Level() <= zap.DebugLevel {
						// this could also happen when the route is synced(deleted) in another syncManifest call,
						// but arrives later than this
						// So log the deleted routes in this call to see if it's true
						if len(deleted.Routes) == 0 {
							log.Debugw("syncManifest deletes upstream but doesn't delete any routes")
						} else {
							found := false

							for _, r := range deleted.Routes {
								if r.UpstreamId == u.ID {
									found = true
									log.Debugw("a deleted route is referencing upstream",
										zap.Any("route", r),
									)
								}
							}
							if !found {
								log.Debugw("no any deleted route is referencing this upstream",
									zap.String("upstream_id", u.ID),
								)
							}
						}

						// try to find which route is referencing the upstream
						routes, err := apisix.Cluster(clusterName).Route().List(ctx)
						if err != nil {
							log.Debugw("try to find referencing routes, but failed to list",
								zap.Error(err),
							)
						}

						found := false
						for _, r := range routes {
							if r.UpstreamId == u.ID {
								found = true
								log.Debugw("route is referencing upstream",
									zap.Any("route", r),
								)
							}
						}
						if !found {
							log.Debugw("failed to find a route that references the upstream",
								zap.String("upstream_id", u.ID),
								zap.Any("routes", routes),
							)
						}
					}
				}
			}
		}
		for _, pc := range deleted.PluginConfigs {
			if err := apisix.Cluster(clusterName).PluginConfig().Delete(ctx, pc); err != nil {
				// pluginConfig might be referenced by other routes.
				if err != cache.ErrStillInUse {
					merr = multierror.Append(merr, err)
				} else {
					log.Infow("plugin_config was referenced by other routes",
						zap.String("plugin_config_id", pc.ID),
						zap.String("plugin_config_name", pc.Name),
					)
				}
			}
		}
		for _, pm := range deleted.PluginMetadatas {
			if err := apisix.Cluster(clusterName).PluginMetadata().Delete(ctx, pm); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
		for _, gr := range deleted.GlobalRules {
			if err := apisix.Cluster(clusterName).GlobalRule().Delete(ctx, gr); err != nil {
				merr = multierror.Append(merr, err)
			}
		}
	}
	if merr != nil {
		return merr
	}
	return nil
}
