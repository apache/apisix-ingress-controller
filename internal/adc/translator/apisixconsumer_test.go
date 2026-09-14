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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func hmacConsumerWithSecret(secretName string) *apiv2.ApisixConsumer {
	return &apiv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: apiv2.ApisixConsumerSpec{
			AuthParameter: &apiv2.ApisixConsumerAuthParameter{
				HMACAuth: &apiv2.ApisixConsumerHMACAuth{
					SecretRef: &corev1.LocalObjectReference{Name: secretName},
				},
			},
		},
	}
}

func hmacSecret(data map[string][]byte) *corev1.Secret {
	data["key_id"] = []byte("my-key")
	data["secret_key"] = []byte("my-secret")
	return &corev1.Secret{Data: data}
}

func TestTranslateApisixConsumer_HMACAuthSignedHeadersFromSecret(t *testing.T) {
	for _, tc := range []struct {
		name     string
		raw      string
		expected []string
	}{
		{name: "comma separated", raw: "X-Date,Host", expected: []string{"X-Date", "Host"}},
		{name: "padding and empty entries", raw: " X-Date, , Host, ", expected: []string{"X-Date", "Host"}},
		{name: "newline separated", raw: "X-Date\nHost", expected: []string{"X-Date", "Host"}},
		{name: "space separated", raw: "X-Date Host", expected: []string{"X-Date", "Host"}},
		{name: "single header", raw: "X-Date", expected: []string{"X-Date"}},
		{name: "empty", raw: "", expected: []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			translator := NewTranslator(logr.Discard(), "")
			tctx := provider.NewDefaultTranslateContext(context.Background())
			tctx.Secrets[k8stypes.NamespacedName{Namespace: "default", Name: "hmac"}] = hmacSecret(map[string][]byte{
				"signed_headers": []byte(tc.raw),
			})

			result, err := translator.TranslateApisixConsumer(tctx, hmacConsumerWithSecret("hmac"))
			require.NoError(t, err)
			require.Len(t, result.Consumers, 1)

			cfg := result.Consumers[0].Plugins["hmac-auth"].(*adctypes.HMACAuthConsumerConfig)
			require.Equal(t, tc.expected, cfg.SignedHeaders)
		})
	}
}

func TestTranslateApisixConsumer_HMACAuthRejectsUnparseableNumbers(t *testing.T) {
	for _, tc := range []struct {
		key string
		raw string
	}{
		{key: "clock_skew", raw: "3O0"}, // typo: letter O
		{key: "max_req_body", raw: "invalid"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			translator := NewTranslator(logr.Discard(), "")
			tctx := provider.NewDefaultTranslateContext(context.Background())
			tctx.Secrets[k8stypes.NamespacedName{Namespace: "default", Name: "hmac"}] = hmacSecret(map[string][]byte{
				tc.key: []byte(tc.raw),
			})

			_, err := translator.TranslateApisixConsumer(tctx, hmacConsumerWithSecret("hmac"))
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.key)
			require.Contains(t, err.Error(), "default/hmac")
		})
	}
}

func TestTranslateApisixConsumer_UsesMetadataLabelsWithoutOverwritingControllerLabels(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())

	consumer := &apiv2.ApisixConsumer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixConsumer",
			APIVersion: apiv2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
			Labels: map[string]string{
				"team":               "payments",
				label.LabelName:      "user-value",
				label.LabelManagedBy: "user-manager",
			},
		},
		Spec: apiv2.ApisixConsumerSpec{
			AuthParameter: &apiv2.ApisixConsumerAuthParameter{
				BasicAuth: &apiv2.ApisixConsumerBasicAuth{
					Value: &apiv2.ApisixConsumerBasicAuthValue{
						Username: "demo",
						Password: "secret",
					},
				},
			},
		},
	}

	result, err := translator.TranslateApisixConsumer(tctx, consumer)
	require.NoError(t, err)
	require.Len(t, result.Consumers, 1)

	translated := result.Consumers[0]
	require.Equal(t, "payments", translated.Labels["team"])
	require.Equal(t, consumer.Name, translated.Labels[label.LabelName])
	require.Equal(t, "apisix-ingress-controller", translated.Labels[label.LabelManagedBy])
}
