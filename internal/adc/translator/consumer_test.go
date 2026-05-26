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
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestTranslateConsumerV1alpha1_UsesMetadataLabelsWithoutOverwritingControllerLabels(t *testing.T) {
	translator := NewTranslator(logr.Discard())
	tctx := provider.NewDefaultTranslateContext(context.Background())

	consumer := &v1alpha1.Consumer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Consumer",
			APIVersion: v1alpha1.GroupVersion.String(),
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
	}

	result, err := translator.TranslateConsumerV1alpha1(tctx, consumer)
	require.NoError(t, err)
	require.Len(t, result.Consumers, 1)

	translated := result.Consumers[0]
	require.Equal(t, "payments", translated.Labels["team"])
	require.Equal(t, consumer.Name, translated.Labels[label.LabelName])
	require.Equal(t, "apisix-ingress-controller", translated.Labels[label.LabelManagedBy])
}
