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

	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

// APISIX is the unified client tool to communicate with APISIX.
type APISIX interface {
	// Cluster specifies the target cluster to talk.
	Cluster(string) Cluster
	// AddCluster adds a new cluster.
	AddCluster(*ClusterOptions) error
	// ListClusters lists all APISIX clusters.
	ListClusters() []Cluster
}

// Cluster defines specific operations that can be applied in an APISIX
// cluster.
type Cluster interface {
	// Route returns a Route interface that can operate Route resources.
	Route() Route
	// Upstream returns a Upstream interface that can operate Upstream resources.
	Upstream() Upstream
	// Service returns a Service interface that can operate Service resources.
	Service() Service
	// SSL returns a SSL interface that can operate SSL resources.
	SSL() SSL
	// String exposes the client information in human readable format.
	String() string
	// Ready waits until all resources in APISIX cluster is synced to
	// cache.
	Ready(context.Context) error
}

// Route is the specific client interface to take over the create, update,
// list and delete for APISIX's Route resource.
type Route interface {
	Get(context.Context, string) (*v1.Route, error)
	List(context.Context) ([]*v1.Route, error)
	Create(context.Context, *v1.Route) (*v1.Route, error)
	Delete(context.Context, *v1.Route) error
	Update(context.Context, *v1.Route) (*v1.Route, error)
}

// SSL is the specific client interface to take over the create, update,
// list and delete for APISIX's SSL resource.
type SSL interface {
	Get(context.Context, string) (*v1.Ssl, error)
	List(context.Context) ([]*v1.Ssl, error)
	Create(context.Context, *v1.Ssl) (*v1.Ssl, error)
	Delete(context.Context, *v1.Ssl) error
	Update(context.Context, *v1.Ssl) (*v1.Ssl, error)
}

// Upstream is the specific client interface to take over the create, update,
// list and delete for APISIX's Upstream resource.
type Upstream interface {
	Get(context.Context, string) (*v1.Upstream, error)
	List(context.Context) ([]*v1.Upstream, error)
	Create(context.Context, *v1.Upstream) (*v1.Upstream, error)
	Delete(context.Context, *v1.Upstream) error
	Update(context.Context, *v1.Upstream) (*v1.Upstream, error)
}

// Service is the specific client interface to take over the create, update,
// list and delete for APISIX's Service resource.
type Service interface {
	Get(context.Context, string) (*v1.Service, error)
	List(context.Context) ([]*v1.Service, error)
	Create(context.Context, *v1.Service) (*v1.Service, error)
	Delete(context.Context, *v1.Service) error
	Update(context.Context, *v1.Service) (*v1.Service, error)
}

type apisix struct {
	nonExistentCluster Cluster
	clusters           map[string]Cluster
}

// New creates an APISIX client to perform resources change pushing.
func New() (APISIX, error) {
	cli := &apisix{
		nonExistentCluster: newNonExistentCluster(),
	}
	return cli, nil
}

// Cluster implements APISIX.Cluster method.
func (c *apisix) Cluster(name string) Cluster {
	cluster, ok := c.clusters[name]
	if !ok {
		return c.nonExistentCluster
	}
	return cluster
}

// ListClusters implements APISIX.ListClusters method.
func (c *apisix) ListClusters() []Cluster {
	clusters := make([]Cluster, 0, len(c.clusters))
	for _, cluster := range c.clusters {
		clusters = append(clusters, cluster)
	}
	return clusters
}

// AddCluster implements APISIX.AddCluster method.
func (c *apisix) AddCluster(co *ClusterOptions) error {
	_, ok := c.clusters[co.Name]
	if ok {
		return ErrDuplicatedCluster
	}
	cluster, err := newCluster(co)
	if err != nil {
		return err
	}
	if c.clusters == nil {
		c.clusters = make(map[string]Cluster)
	}
	c.clusters[co.Name] = cluster
	return nil
}
