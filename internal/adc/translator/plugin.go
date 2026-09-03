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
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

// renderPluginConfig renders the configuration of an apisix.apache.org/v1alpha1 Plugin.
// The data of the referenced Secret is merged over spec.config, with each Secret key
// read as a dot separated path so that `session.secret` nests under `session`.
func renderPluginConfig(plugin v1alpha1.Plugin, namespace string, secrets map[types.NamespacedName]*corev1.Secret) (map[string]any, error) {
	config := make(map[string]any)
	if len(plugin.Config.Raw) > 0 {
		if err := json.Unmarshal(plugin.Config.Raw, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config of plugin %s: %w", plugin.Name, err)
		}
	}
	// A literal `config: null` unmarshals to a nil map, which serializes back to
	// null and is rejected by most APISIX plugins; normalize it to an empty object.
	if config == nil {
		config = make(map[string]any)
	}
	if plugin.SecretRef == nil || plugin.SecretRef.Name == "" {
		return config, nil
	}
	secret, ok := secrets[types.NamespacedName{Namespace: namespace, Name: plugin.SecretRef.Name}]
	if !ok || secret == nil {
		return nil, fmt.Errorf("secret %s/%s referenced by plugin %s not found", namespace, plugin.SecretRef.Name, plugin.Name)
	}
	for key, value := range secret.Data {
		pkgutils.InsertKeyInMap(key, string(value), config)
	}
	return config, nil
}
