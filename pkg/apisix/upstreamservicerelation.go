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
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

// There is no need to ensure the consistency between the upstream to services, only need to ensure that the upstream-node can be delete after deleting the service
type upstreamService struct {
	cluster *cluster
}

func newUpstreamServiceRelation(c *cluster) *upstreamService {
	return &upstreamService{
		cluster: c,
	}
}

func (u *upstreamService) Get(ctx context.Context, svcId string) (*v1.UpstreamServiceRelation, error) {
	log.Debugw("try to get upstreamService in cache",
		zap.String("svcId", svcId),
		zap.String("cluster", "default"),
	)
	us, err := u.cluster.cache.GetUpstreamServiceRelation(svcId)
	if err == nil {
		return us, nil
	}
	if err != cache.ErrNotFound {
		log.Error("failed to find upstreamService in cache",
			zap.String("svcId", svcId), zap.Error(err))
	} else {
		log.Debugw("failed to find upstreamService in cache",
			zap.String("svcId", svcId), zap.Error(err))
	}
	return nil, err
}

func (u *upstreamService) Delete(ctx context.Context, relation *v1.UpstreamServiceRelation) error {
	log.Debugw("try to delete upstreamService in cache",
		zap.String("cluster", "default"),
	)
	u.initUpstreamServiceRelation(relation)
	if relation == nil || relation.ServiceName == "" && relation.UpstreamName == "" {
		return fmt.Errorf("UpstreamServiceRelation is empty object")
	}
	if relation.UpstreamName != "" {
		err := u.cluster.cache.DeleteUpstreamServiceRelation(relation)
		if err != nil {
			return err
		}
	} else {
		usr, err := u.cluster.cache.GetUpstreamServiceRelation(relation.ServiceName)
		if err != nil {
			return err
		}
		ups, err := u.cluster.upstream.Get(ctx, usr.UpstreamName)
		if err != nil {
			return err
		}
		ups.Nodes = make(v1.UpstreamNodes, 0)
		_, err = u.cluster.upstream.Update(ctx, ups)
		if err != nil {
			return err
		}
		err = u.cluster.cache.DeleteUpstreamServiceRelation(usr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *upstreamService) Create(ctx context.Context, relation *v1.UpstreamServiceRelation) error {
	log.Debugw("try to create upstreamService in cache",
		zap.String("cluster", "default"),
	)
	u.initUpstreamServiceRelation(relation)
	if relation == nil || relation.ServiceName == "" || relation.UpstreamName == "" {
		log.Error("UpstreamServiceRelation object ")
		return nil
	}
	us, err := u.cluster.cache.GetUpstreamServiceRelation(relation.ServiceName)
	if err != nil && err != cache.ErrNotFound {
		return err
	}
	if us != nil {
		us.UpstreamName = relation.UpstreamName
	} else {
		us = relation
	}
	if err := u.cluster.cache.InsertUpstreamServiceRelation(us); err != nil {
		log.Errorf("failed to reflect upstreamService create to cache: %s", err)
		return err
	}
	return nil
}

func (u *upstreamService) List(ctx context.Context) ([]*v1.UpstreamServiceRelation, error) {
	log.Debugw("try to create upstreamService in cache",
		zap.String("cluster", "default"),
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

func (u *upstreamService) initUpstreamServiceRelation(us *v1.UpstreamServiceRelation) {
	if us.UpstreamName == "" || us.ServiceName != "" {
		return
	}
	args := strings.Split(us.UpstreamName, "_")
	// namespace_service_subset_port
	if len(args) < 2 {
		return
	}
	us.ServiceName = args[0] + "_" + args[1]
}
