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

package client

import (
	"testing"

	"github.com/stretchr/testify/require"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
)

func TestBuildAPISIXValidatePayloadConvertsSSLCertificates(t *testing.T) {
	body, err := buildAPISIXValidatePayload(&adctypes.Resources{
		SSLs: []*adctypes.SSL{
			{
				Metadata: adctypes.Metadata{ID: "ssl-1"},
				Snis:     []string{"example.com"},
				Certificates: []adctypes.Certificate{
					{
						Certificate: "leaf-cert",
						Key:         "leaf-key",
					},
					{
						Certificate: "chain-cert",
						Key:         "chain-key",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, body.SSLs, 1)

	ssl := body.SSLs[0]
	require.Equal(t, "ssl-1", ssl["id"])
	require.Equal(t, "leaf-cert", ssl["cert"])
	require.Equal(t, "leaf-key", ssl["key"])
	require.Equal(t, []string{"chain-cert"}, ssl["certs"])
	require.Equal(t, []string{"chain-key"}, ssl["keys"])
	_, ok := ssl["certificates"]
	require.False(t, ok)
}

func TestBuildAPISIXValidatePayloadConvertsSingleSSLCertificate(t *testing.T) {
	body, err := buildAPISIXValidatePayload(&adctypes.Resources{
		SSLs: []*adctypes.SSL{
			{
				Metadata: adctypes.Metadata{ID: "ssl-1"},
				Snis:     []string{"example.com"},
				Certificates: []adctypes.Certificate{
					{
						Certificate: "leaf-cert",
						Key:         "leaf-key",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, body.SSLs, 1)

	ssl := body.SSLs[0]
	require.Equal(t, "leaf-cert", ssl["cert"])
	require.Equal(t, "leaf-key", ssl["key"])
	_, ok := ssl["certs"]
	require.False(t, ok)
	_, ok = ssl["keys"]
	require.False(t, ok)
}

func TestBuildAPISIXValidatePayloadStripsRouteDescription(t *testing.T) {
	body, err := buildAPISIXValidatePayload(&adctypes.Resources{
		Services: []*adctypes.Service{
			{
				Metadata: adctypes.Metadata{ID: "svc-1"},
				Routes: []*adctypes.Route{
					{
						Metadata: adctypes.Metadata{
							ID:   "route-1",
							Desc: "should not be sent to standalone validate",
						},
						Uris: []string{"/test"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, body.Routes, 1)

	route := body.Routes[0]
	require.Equal(t, "route-1", route["id"])
	_, ok := route["description"]
	require.False(t, ok)
	require.Equal(t, "svc-1", route["service_id"])
}

func TestBuildAPISIXValidatePayloadConvertsConsumerCredentialsToPlugins(t *testing.T) {
	body, err := buildAPISIXValidatePayload(&adctypes.Resources{
		Consumers: []*adctypes.Consumer{
			{
				Metadata: adctypes.Metadata{ID: "consumer-1"},
				Username: "demo",
				Plugins: adctypes.Plugins{
					"limit-count": map[string]any{"count": 10},
				},
				Credentials: []adctypes.Credential{
					{
						Type: "key-auth",
						Config: adctypes.Plugins{
							"key": "shared-key",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, body.Consumers, 1)

	consumer := body.Consumers[0]
	require.Equal(t, "demo", consumer["username"])
	_, ok := consumer["credentials"]
	require.False(t, ok)

	plugins, ok := consumer["plugins"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, plugins, "key-auth")
	require.Contains(t, plugins, "limit-count")
}
