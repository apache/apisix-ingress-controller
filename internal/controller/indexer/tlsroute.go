// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func setupTLSRouteIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1alpha2.TLSRoute{},
		ParentRefs,
		TLSRouteParentRefsIndexFunc,
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1alpha2.TLSRoute{},
		ServiceIndexRef,
		TLSPRouteServiceIndexFunc,
	); err != nil {
		return err
	}
	return nil
}

func TLSRouteParentRefsIndexFunc(rawObj client.Object) []string {
	tr := rawObj.(*gatewayv1alpha2.TLSRoute)
	keys := make([]string, 0, len(tr.Spec.ParentRefs))
	for _, ref := range tr.Spec.ParentRefs {
		ns := tr.GetNamespace()
		if ref.Namespace != nil {
			ns = string(*ref.Namespace)
		}
		keys = append(keys, GenIndexKey(ns, string(ref.Name)))
	}
	return keys
}

func TLSPRouteServiceIndexFunc(rawObj client.Object) []string {
	tr := rawObj.(*gatewayv1alpha2.TLSRoute)
	keys := make([]string, 0, len(tr.Spec.Rules))
	for _, rule := range tr.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			namespace := tr.GetNamespace()
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
