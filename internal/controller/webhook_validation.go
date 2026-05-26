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

package controller

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func PrepareApisixRouteForValidation(ctx context.Context, c client.Client, log logr.Logger, route *apiv2.ApisixRoute) (*provider.TranslateContext, error) {
	tctx := provider.NewDefaultTranslateContext(ctx)

	ingressClass, err := FindMatchingIngressClass(tctx, c, log, route)
	if err != nil {
		return nil, err
	}
	if err := ProcessIngressClassParameters(tctx, c, log, route, ingressClass); err != nil {
		return nil, err
	}

	reconciler := &ApisixRouteReconciler{
		Client: c,
		Log:    log,
	}
	if err := reconciler.processApisixRoute(tctx, route); err != nil {
		return nil, err
	}
	return tctx, nil
}

func PrepareApisixConsumerForValidation(ctx context.Context, c client.Client, log logr.Logger, consumer *apiv2.ApisixConsumer) (*provider.TranslateContext, error) {
	tctx := provider.NewDefaultTranslateContext(ctx)

	ingressClass, err := FindMatchingIngressClass(tctx, c, log, consumer)
	if err != nil {
		return nil, err
	}
	if err := ProcessIngressClassParameters(tctx, c, log, consumer, ingressClass); err != nil {
		return nil, err
	}

	reconciler := &ApisixConsumerReconciler{
		Client: c,
		Log:    log,
	}
	if err := reconciler.processSpec(ctx, tctx, consumer); err != nil {
		return nil, err
	}
	return tctx, nil
}

func PrepareConsumerForValidation(ctx context.Context, c client.Client, log logr.Logger, consumer *v1alpha1.Consumer) (*provider.TranslateContext, error) {
	tctx := provider.NewDefaultTranslateContext(ctx)

	reconciler := &ConsumerReconciler{
		Client: c,
		Log:    log,
	}
	gateway, err := reconciler.getGateway(ctx, consumer)
	if err != nil {
		return nil, err
	}
	if err := ProcessGatewayProxy(c, log, tctx, gateway, utils.NamespacedNameKind(consumer)); err != nil {
		return nil, err
	}
	if err := reconciler.processSpec(ctx, tctx, consumer); err != nil {
		return nil, err
	}
	return tctx, nil
}

func PrepareApisixTlsForValidation(ctx context.Context, c client.Client, log logr.Logger, tls *apiv2.ApisixTls) (*provider.TranslateContext, error) {
	tctx := provider.NewDefaultTranslateContext(ctx)

	ingressClass, err := FindMatchingIngressClass(tctx, c, log, tls)
	if err != nil {
		return nil, err
	}
	if err := ProcessIngressClassParameters(tctx, c, log, tls, ingressClass); err != nil {
		return nil, err
	}

	reconciler := &ApisixTlsReconciler{
		Client: c,
		Log:    log,
	}
	if err := reconciler.processApisixTls(ctx, tctx, tls); err != nil {
		return nil, err
	}
	return tctx, nil
}
