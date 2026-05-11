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

package label

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

func TestGenLabelWithObjectLabels(t *testing.T) {
	consumer := &apiv2.ApisixConsumer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixConsumer",
			APIVersion: apiv2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
			Labels: map[string]string{
				"team":              "payments",
				LabelName:           "user-value",
				LabelManagedBy:      "user-manager",
				LabelNamespace:      "user-namespace",
				LabelControllerName: "user-controller",
				LabelKind:           "user-kind",
			},
		},
	}

	labels := GenLabelWithObjectLabels(consumer)

	require.Equal(t, "payments", labels["team"])
	require.Equal(t, consumer.Name, labels[LabelName])
	require.Equal(t, consumer.Namespace, labels[LabelNamespace])
	require.Equal(t, "ApisixConsumer", labels[LabelKind])
	require.Equal(t, config.ControllerConfig.ControllerName, labels[LabelControllerName])
	require.Equal(t, "apisix-ingress-controller", labels[LabelManagedBy])
}

func TestGenLabel_IgnoresDanglingKeyArg(t *testing.T) {
	consumer := &apiv2.ApisixConsumer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixConsumer",
			APIVersion: apiv2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
	}

	labels := GenLabel(consumer, "team", "payments", "dangling")

	require.Equal(t, "payments", labels["team"])
	require.NotContains(t, labels, "dangling")
}
