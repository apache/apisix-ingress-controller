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

package apisix

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// TestDeleteNotifiesSyncOnlyWhenConfigWasRemoved covers the cost side of route
// ownership: a sync pushes the whole store to every data plane, and reconciles
// for routes this controller never configured are frequent (any EndpointSlice
// event on a shared backend enqueues them), so those must not notify.
func TestDeleteNotifiesSyncOnlyWhenConfigWasRemoved(t *testing.T) {
	cli, err := adcclient.New(logr.Discard(), ProviderTypeAPISIX, time.Second)
	require.NoError(t, err)

	d := &apisixProvider{
		client: cli,
		syncCh: make(chan struct{}, 1),
		log:    logr.Discard(),
	}

	route := &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPRoute",
			APIVersion: gatewayv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
	}

	require.NoError(t, d.Delete(context.Background(), route))
	require.Empty(t, d.syncCh, "a route this controller never configured must not trigger a sync")

	cli.ConfigManager.Update(utils.NamespacedNameKind(route), map[types.NamespacedNameKind]adctypes.Config{
		{Namespace: "default", Name: "proxy", Kind: "GatewayProxy"}: {Name: "proxy"},
	})

	require.NoError(t, d.Delete(context.Background(), route))
	require.Len(t, d.syncCh, 1, "removing configuration this controller pushed must trigger a sync")
}
