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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

func loadPluginConfigExtensionRef(
	ctx context.Context,
	c client.Client,
	tctx *provider.TranslateContext,
	namespace string,
	ref *gatewayv1.LocalObjectReference,
) error {
	if err := types.ValidatePluginConfigExtensionRef(ref); err != nil {
		return err
	}

	pluginConfig := new(v1alpha1.PluginConfig)
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      string(ref.Name),
	}, pluginConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return types.NewPluginConfigNotFoundError(namespace, string(ref.Name))
		}
		return err
	}

	tctx.PluginConfigs[k8stypes.NamespacedName{
		Namespace: namespace,
		Name:      string(ref.Name),
	}] = pluginConfig
	if err := loadPluginSecrets(ctx, c, tctx, namespace, pluginConfig.Spec.Plugins); err != nil {
		if apierrors.IsNotFound(err) {
			return types.ReasonError{
				Reason:  string(gatewayv1.RouteReasonBackendNotFound),
				Message: err.Error(),
			}
		}
		return err
	}
	return nil
}
