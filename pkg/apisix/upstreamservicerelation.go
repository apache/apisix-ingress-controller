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
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

// to do: Delete one of the upstreams. Currently, only service is deleted. There will be some
// redundant upstream objects, but the results will not be affected. It is hoped that  the service controller
// can complete the update nodes logic to avoid the intrusion of relation modules into more code.

// Maintain relationships only when resolveGranularity is endpoint
// There is no need to ensure the consistency between the upstream to services, only need to ensure that the upstream-node can be delete after deleting the service
type upstreamService struct {
	cluster *cluster
}

func newUpstreamServiceRelation(c *cluster) *upstreamService {
	return &upstreamService{
		cluster: c,
	}
}

func (u *upstreamService) Get(ctx context.Context, serviceName string) (*v1.UpstreamServiceRelation, error) {
	log.Debugw("try to get upstreamService in cache",
		zap.String("service_name", serviceName),
		zap.String("cluster", u.cluster.name),
	)
	us, err := u.cluster.cache.GetUpstreamServiceRelation(serviceName)
	if err != nil {
		log.Error("failed to find upstreamService in cache",
			zap.String("service_name", serviceName), zap.Error(err))
		return nil, err
	}
	return us, err
}

func (u *upstreamService) Delete(ctx context.Context, serviceName string) error {
	log.Debugw("try to delete upstreamService in cache",
		zap.String("cluster", u.cluster.name),
	)
	relation, err := u.Get(ctx, serviceName)
	if err != nil {
		if err == cache.ErrNotFound {
			return nil
		}
		return err
	}
	_ = u.cluster.cache.DeleteUpstreamServiceRelation(relation)
	for upsName := range relation.UpstreamNames {
		ups, err := u.cluster.upstream.Get(ctx, upsName)
		if err != nil || ups == nil {
			continue
		}
		ups.Nodes = make(v1.UpstreamNodes, 0)
		log.Debugw("try to update upstream in cluster",
			zap.Any("upstream", ups),
		)
		_, err = u.cluster.upstream.Update(ctx, ups, false)
		if err != nil {
			log.Error(err)
			continue
		}
	}
	return nil
}

func (u *upstreamService) Create(ctx context.Context, upstreamName string) error {
	log.Debugw("try to create upstreamService in cache",
		zap.String("cluster", u.cluster.name),
	)

	args := strings.Split(upstreamName, "_")
	if len(args) < 2 {
		return fmt.Errorf("wrong upstream name %s, must contains namespace_name", upstreamName)
	}
	// The last part of upstreanName should be a port number.
	// Please refer to apisixv1.ComposeUpstreamName to see the detailed format.
	_, err := strconv.Atoi(args[len(args)-1])
	if err != nil {
		return nil
	}

	serviceName := args[0] + "_" + args[1]
	relation, err := u.Get(ctx, serviceName)
	if err != nil {
		return err
	}
	if relation == nil {
		relation = &v1.UpstreamServiceRelation{
			ServiceName: serviceName,
			UpstreamNames: map[string]struct{}{
				upstreamName: {},
			},
		}
	} else {
		relation.UpstreamNames[upstreamName] = struct{}{}
	}
	if err := u.cluster.cache.InsertUpstreamServiceRelation(relation); err != nil {
		log.Errorf("failed to reflect upstreamService create to cache: %s", err)
		return err
	}
	return nil
}

func (u *upstreamService) List(ctx context.Context) ([]*v1.UpstreamServiceRelation, error) {
	log.Debugw("try to create upstreamService in cache",
		zap.String("cluster", u.cluster.name),
	)
	usrs, err := u.cluster.cache.ListUpstreamServiceRelation()
	if err != nil {
		log.Errorw("failed to list upstream in cache",
			zap.Error(err),
		)
		return nil, err
	}
	return usrs, nil
}
