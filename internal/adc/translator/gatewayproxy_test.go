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

package translator

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestTranslateGatewayProxyToConfig_TlsVerifyDefault(t *testing.T) {
	newProxy := func(tlsVerify *bool) *v1alpha1.GatewayProxy {
		return &v1alpha1.GatewayProxy{
			ObjectMeta: metav1.ObjectMeta{Name: "gp", Namespace: "default"},
			Spec: v1alpha1.GatewayProxySpec{
				Provider: &v1alpha1.GatewayProxyProvider{
					Type: v1alpha1.ProviderTypeControlPlane,
					ControlPlane: &v1alpha1.ControlPlaneProvider{
						Endpoints: []string{"https://127.0.0.1:7443"},
						TlsVerify: tlsVerify,
						Auth: v1alpha1.ControlPlaneAuth{
							Type:     v1alpha1.AuthTypeAdminKey,
							AdminKey: &v1alpha1.AdminKeyAuth{Value: "secret"},
						},
					},
				},
			},
		}
	}

	tr := false
	tt := true
	cases := []struct {
		name      string
		tlsVerify *bool
		want      bool
	}{
		{"unset defaults to verify", nil, true},
		{"explicit false opts out", &tr, false},
		{"explicit true verifies", &tt, true},
	}

	translator := NewTranslator(logr.Discard(), "")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tctx := provider.NewDefaultTranslateContext(context.Background())
			cfg, err := translator.TranslateGatewayProxyToConfig(tctx, newProxy(c.tlsVerify), false)
			require.NoError(t, err)
			require.NotNil(t, cfg)
			require.Equal(t, c.want, cfg.TlsVerify)
		})
	}
}
