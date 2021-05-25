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
	"bytes"
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type consumerClient struct {
	url     string
	cluster *cluster
}

func newConsumerClient(c *cluster) Consumer {
	return &consumerClient{
		url:     c.baseURL + "/consumers",
		cluster: c,
	}
}

// Get returns the Consumer.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (r *consumerClient) Get(ctx context.Context, name string) (*v1.Consumer, error) {
	log.Debugw("try to look up consumer",
		zap.String("name", name),
		zap.String("url", r.url),
		zap.String("cluster", "default"),
	)
	consumer, err := r.cluster.cache.GetConsumer(name)
	if err == nil {
		return consumer, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find consumer in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	} else {
		log.Debugw("consumer not found in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effect.
	url := r.url + "/" + name
	resp, err := r.cluster.getResource(ctx, url)
	if err != nil {
		if err == cache.ErrNotFound {
			log.Warnw("consumer not found",
				zap.String("name", name),
				zap.String("url", url),
				zap.String("cluster", "default"),
			)
		} else {
			log.Errorw("failed to get consumer from APISIX",
				zap.String("name", name),
				zap.String("url", url),
				zap.String("cluster", "default"),
				zap.Error(err),
			)
		}
		return nil, err
	}

	consumer, err = resp.Item.consumer()
	if err != nil {
		log.Errorw("failed to convert consumer item",
			zap.String("url", r.url),
			zap.String("consumer_key", resp.Item.Key),
			zap.String("consumer_value", string(resp.Item.Value)),
			zap.Error(err),
		)
		return nil, err
	}

	if err := r.cluster.cache.InsertConsumer(consumer); err != nil {
		log.Errorf("failed to reflect consumer create to cache: %s", err)
		return nil, err
	}
	return consumer, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *consumerClient) List(ctx context.Context) ([]*v1.Consumer, error) {
	log.Debugw("try to list consumers in APISIX",
		zap.String("cluster", "default"),
		zap.String("url", r.url),
	)
	consumerItems, err := r.cluster.listResource(ctx, r.url)
	if err != nil {
		log.Errorf("failed to list consumers: %s", err)
		return nil, err
	}

	var items []*v1.Consumer
	for i, item := range consumerItems.Node.Items {
		consumer, err := item.consumer()
		if err != nil {
			log.Errorw("failed to convert consumer item",
				zap.String("url", r.url),
				zap.String("consumer_key", item.Key),
				zap.String("consumer_value", string(item.Value)),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, consumer)
		log.Debugf("list consumer #%d, body: %s", i, string(item.Value))
	}

	return items, nil
}

func (r *consumerClient) Create(ctx context.Context, obj *v1.Consumer) (*v1.Consumer, error) {
	log.Debugw("try to create consumer",
		zap.String("name", obj.Username),
		zap.Any("plugins", obj.Plugins),
		zap.String("cluster", "default"),
		zap.String("url", r.url),
	)

	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	url := r.url + "/" + obj.Username
	log.Debugw("creating consumer", zap.ByteString("body", data), zap.String("url", url))
	resp, err := r.cluster.createResource(ctx, url, bytes.NewReader(data))
	if err != nil {
		log.Errorf("failed to create consumer: %s", err)
		return nil, err
	}

	consumer, err := resp.Item.consumer()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertConsumer(consumer); err != nil {
		log.Errorf("failed to reflect consumer create to cache: %s", err)
		return nil, err
	}
	return consumer, nil
}

func (r *consumerClient) Delete(ctx context.Context, obj *v1.Consumer) error {
	log.Debugw("try to delete consumer",
		zap.String("name", obj.Username),
		zap.String("cluster", "default"),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.Username
	if err := r.cluster.deleteResource(ctx, url); err != nil {
		return err
	}
	if err := r.cluster.cache.DeleteConsumer(obj); err != nil {
		log.Errorf("failed to reflect consumer delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (r *consumerClient) Update(ctx context.Context, obj *v1.Consumer) (*v1.Consumer, error) {
	log.Debugw("try to update consumer",
		zap.String("name", obj.Username),
		zap.Any("plugins", obj.Plugins),
		zap.String("cluster", "default"),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	url := r.url + "/" + obj.Username
	log.Debugw("updating username", zap.ByteString("body", body), zap.String("url", url))
	resp, err := r.cluster.updateResource(ctx, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	consumer, err := resp.Item.consumer()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertConsumer(consumer); err != nil {
		log.Errorf("failed to reflect consumer update to cache: %s", err)
		return nil, err
	}
	return consumer, nil
}
