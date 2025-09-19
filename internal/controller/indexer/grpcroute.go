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

package indexer

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func setupGRPCRouteIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.GRPCRoute{},
		ParentRefs,
		GRPCRouteParentRefsIndexFunc,
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.GRPCRoute{},
		ExtensionRef,
		GRPCRouteExtensionIndexFunc,
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.GRPCRoute{},
		ServiceIndexRef,
		GRPCRouteServiceIndexFunc,
	); err != nil {
		return err
	}

	return nil
}

func GRPCRouteParentRefsIndexFunc(rawObj client.Object) []string {
	gr := rawObj.(*gatewayv1.GRPCRoute)
	keys := make([]string, 0, len(gr.Spec.ParentRefs))
	for _, ref := range gr.Spec.ParentRefs {
		ns := gr.GetNamespace()
		if ref.Namespace != nil {
			ns = string(*ref.Namespace)
		}
		keys = append(keys, GenIndexKey(ns, string(ref.Name)))
	}
	return keys
}

func GRPCRouteServiceIndexFunc(rawObj client.Object) []string {
	gr := rawObj.(*gatewayv1.GRPCRoute)
	keys := make([]string, 0, len(gr.Spec.Rules))
	for _, rule := range gr.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			namespace := gr.GetNamespace()
			if backend.Kind != nil && *backend.Kind != internaltypes.KindService {
				continue
			}
			if backend.Namespace != nil {
				namespace = string(*backend.Namespace)
			}
			keys = append(keys, GenIndexKey(namespace, string(backend.Name)))
		}
	}
	return keys
}

func GRPCRouteExtensionIndexFunc(rawObj client.Object) []string {
	gr := rawObj.(*gatewayv1.GRPCRoute)
	keys := make([]string, 0, len(gr.Spec.Rules))
	for _, rule := range gr.Spec.Rules {
		for _, filter := range rule.Filters {
			if filter.Type != gatewayv1.GRPCRouteFilterExtensionRef || filter.ExtensionRef == nil {
				continue
			}
			if filter.ExtensionRef.Kind == internaltypes.KindPluginConfig {
				keys = append(keys, GenIndexKey(gr.GetNamespace(), string(filter.ExtensionRef.Name)))
			}
		}
	}
	return keys
}
