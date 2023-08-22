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

package apisix

import (
	"context"
	"sync"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

// APISIX is the unified client tool to communicate with APISIX.
type APISIX interface {
	// Cluster specifies the target cluster to talk.
	Cluster(name string) Cluster
	// AddCluster adds a new cluster.
	AddCluster(context.Context, *ClusterOptions) error
	// UpdateCluster updates an existing cluster.
	UpdateCluster(context.Context, *ClusterOptions) error
	// ListClusters lists all APISIX clusters.
	ListClusters() []Cluster
	// DeleteCluster deletes the target APISIX cluster by its name.
	DeleteCluster(name string)
}

// Cluster defines specific operations that can be applied in an APISIX
// cluster.
type Cluster interface {
	// Route returns a Route interface that can operate Route resources.
	Route() Route
	// Upstream returns a Upstream interface that can operate Upstream resources.
	Upstream() Upstream
	// SSL returns a SSL interface that can operate SSL resources.
	SSL() SSL
	// StreamRoute returns a StreamRoute interface that can operate StreamRoute resources.
	StreamRoute() StreamRoute
	// GlobalRule returns a GlobalRule interface that can operate GlobalRule resources.
	GlobalRule() GlobalRule
	// String exposes the client information in human-readable format.
	String() string
	// HasSynced checks whether all resources in APISIX cluster is synced to cache.
	HasSynced(context.Context) error
	// Consumer returns a Consumer interface that can operate Consumer resources.
	Consumer() Consumer
	// HealthCheck checks apisix cluster health in realtime.
	HealthCheck(context.Context) error
	// Plugin returns a Plugin interface that can operate Plugin resources.
	Plugin() Plugin
	// PluginConfig returns a PluginConfig interface that can operate PluginConfig resources.
	PluginConfig() PluginConfig
	// Schema returns a Schema interface that can fetch schema of APISIX objects.
	Schema() Schema

	PluginMetadata() PluginMetadata
	// UpstreamServiceRelation returns a UpstreamServiceRelation interface that can fetch UpstreamServiceRelation of APISIX objects.
	UpstreamServiceRelation() UpstreamServiceRelation

	Validator() APISIXSchemaValidator
}

// Route is the specific client interface to take over the create, update,
// list and delete for APISIX Route resource.
type Route interface {
	Get(ctx context.Context, name string) (*v1.Route, error)
	List(ctx context.Context) ([]*v1.Route, error)
	Create(ctx context.Context, route *v1.Route, shouldCompare bool) (*v1.Route, error)
	Delete(ctx context.Context, route *v1.Route) error
	Update(ctx context.Context, route *v1.Route, shouldCompare bool) (*v1.Route, error)
}

// SSL is the specific client interface to take over the create, update,
// list and delete for APISIX SSL resource.
type SSL interface {
	// name is namespace_sslname
	Get(ctx context.Context, name string) (*v1.Ssl, error)
	List(ctx context.Context) ([]*v1.Ssl, error)
	Create(ctx context.Context, ssl *v1.Ssl, shouldCompare bool) (*v1.Ssl, error)
	Delete(ctx context.Context, ssl *v1.Ssl) error
	Update(ctx context.Context, ssl *v1.Ssl, shouldCompare bool) (*v1.Ssl, error)
}

// Upstream is the specific client interface to take over the create, update,
// list and delete for APISIX Upstream resource.
type Upstream interface {
	Get(ctx context.Context, name string) (*v1.Upstream, error)
	List(ctx context.Context) ([]*v1.Upstream, error)
	Create(ctx context.Context, ups *v1.Upstream, shouldCompare bool) (*v1.Upstream, error)
	Delete(ctx context.Context, ups *v1.Upstream) error
	Update(ctx context.Context, ups *v1.Upstream, shouldCompare bool) (*v1.Upstream, error)
}

// StreamRoute is the specific client interface to take over the create, update,
// list and delete for APISIX Stream Route resource.
type StreamRoute interface {
	Get(ctx context.Context, name string) (*v1.StreamRoute, error)
	List(ctx context.Context) ([]*v1.StreamRoute, error)
	Create(ctx context.Context, route *v1.StreamRoute, shouldCompare bool) (*v1.StreamRoute, error)
	Delete(ctx context.Context, route *v1.StreamRoute) error
	Update(ctx context.Context, route *v1.StreamRoute, shouldCompare bool) (*v1.StreamRoute, error)
}

// GlobalRule is the specific client interface to take over the create, update,
// list and delete for APISIX Global Rule resource.
type GlobalRule interface {
	Get(ctx context.Context, id string) (*v1.GlobalRule, error)
	List(ctx context.Context) ([]*v1.GlobalRule, error)
	Create(ctx context.Context, rule *v1.GlobalRule, shouldCompare bool) (*v1.GlobalRule, error)
	Delete(ctx context.Context, rule *v1.GlobalRule) error
	Update(ctx context.Context, rule *v1.GlobalRule, shouldCompare bool) (*v1.GlobalRule, error)
}

// Consumer is the specific client interface to take over the create, update,
// list and delete for APISIX Consumer resource.
type Consumer interface {
	Get(ctx context.Context, name string) (*v1.Consumer, error)
	List(ctx context.Context) ([]*v1.Consumer, error)
	Create(ctx context.Context, consumer *v1.Consumer, shouldCompare bool) (*v1.Consumer, error)
	Delete(ctx context.Context, consumer *v1.Consumer) error
	Update(ctx context.Context, consumer *v1.Consumer, shouldCompare bool) (*v1.Consumer, error)
}

// Plugin is the specific client interface to fetch APISIX Plugin resource.
type Plugin interface {
	List(ctx context.Context) ([]string, error)
}

// Schema is the specific client interface to fetch the schema of APISIX objects.
type Schema interface {
	GetPluginSchema(ctx context.Context, pluginName string) (*v1.Schema, error)
	GetRouteSchema(ctx context.Context) (*v1.Schema, error)
	GetUpstreamSchema(ctx context.Context) (*v1.Schema, error)
	GetConsumerSchema(ctx context.Context) (*v1.Schema, error)
	GetSslSchema(ctx context.Context) (*v1.Schema, error)
	GetPluginConfigSchema(ctx context.Context) (*v1.Schema, error)
}

// PluginConfig is the specific client interface to take over the create, update,
// list and delete for APISIX PluginConfig resource.
type PluginConfig interface {
	Get(ctx context.Context, name string) (*v1.PluginConfig, error)
	List(ctx context.Context) ([]*v1.PluginConfig, error)
	Create(ctx context.Context, plugin *v1.PluginConfig, shouldCompare bool) (*v1.PluginConfig, error)
	Delete(ctx context.Context, plugin *v1.PluginConfig) error
	Update(ctx context.Context, plugin *v1.PluginConfig, shouldCompare bool) (*v1.PluginConfig, error)
}

type PluginMetadata interface {
	Get(ctx context.Context, name string) (*v1.PluginMetadata, error)
	List(ctx context.Context) ([]*v1.PluginMetadata, error)
	Delete(ctx context.Context, metadata *v1.PluginMetadata) error
	Update(ctx context.Context, metadata *v1.PluginMetadata, shouldCompare bool) (*v1.PluginMetadata, error)
	Create(ctx context.Context, metadata *v1.PluginMetadata, shouldCompare bool) (*v1.PluginMetadata, error)
}

type UpstreamServiceRelation interface {
	// Get relation based on namespace+"_"+service.name
	Get(ctx context.Context, svcName string) (*v1.UpstreamServiceRelation, error)
	List(ctx context.Context) ([]*v1.UpstreamServiceRelation, error)
	// Delete relation based on namespace+"_"+service.name
	Delete(ctx context.Context, svcName string) error
	// Build relation based on upstream.name
	Create(ctx context.Context, svcName string) error
}

type APISIXSchemaValidator interface {
	ValidateSteamPluginSchema(plugins v1.Plugins) (bool, error)
	ValidateHTTPPluginSchema(plugins v1.Plugins) (bool, error)
}

type apisix struct {
	adminVersion       string
	mu                 sync.RWMutex
	nonExistentCluster Cluster
	clusters           map[string]Cluster
}

// NewClient creates an APISIX client to perform resources change pushing.
func NewClient(version string) (APISIX, error) {
	cli := &apisix{
		adminVersion:       version,
		nonExistentCluster: newNonExistentCluster(),
		clusters:           make(map[string]Cluster),
	}
	return cli, nil
}

// Cluster implements APISIX.Cluster method.
func (c *apisix) Cluster(name string) Cluster {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cluster, ok := c.clusters[name]
	if !ok {
		return c.nonExistentCluster
	}
	return cluster
}

// ListClusters implements APISIX.ListClusters method.
func (c *apisix) ListClusters() []Cluster {
	c.mu.RLock()
	defer c.mu.RUnlock()
	clusters := make([]Cluster, 0, len(c.clusters))
	for _, cluster := range c.clusters {
		clusters = append(clusters, cluster)
	}
	return clusters
}

// AddCluster implements APISIX.AddCluster method.
func (c *apisix) AddCluster(ctx context.Context, co *ClusterOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.clusters[co.Name]
	if ok {
		return ErrDuplicatedCluster
	}
	if co.AdminAPIVersion == "" {
		co.AdminAPIVersion = c.adminVersion
	}
	cluster, err := newCluster(ctx, co)
	if err != nil {
		return err
	}
	c.clusters[co.Name] = cluster
	return nil
}

func (c *apisix) UpdateCluster(ctx context.Context, co *ClusterOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.clusters[co.Name]; !ok {
		return ErrClusterNotExist
	}

	if co.AdminAPIVersion == "" {
		co.AdminAPIVersion = c.adminVersion
	}
	cluster, err := newCluster(ctx, co)
	if err != nil {
		return err
	}

	c.clusters[co.Name] = cluster
	return nil
}

func (c *apisix) DeleteCluster(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Don't have to close or free some resources in that cluster, so
	// just delete its index.
	delete(c.clusters, name)
}
