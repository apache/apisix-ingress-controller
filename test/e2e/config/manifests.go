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
package config

const (
	_ingressAPISIXConfigMapTemplate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: ingress-apisix-controller-config
data:
  config.yaml: |
    apisix:
      default_cluster_base_url: "{{.DEFAULT_CLUSTER_BASE_URL}}"
      default_cluster_admin_key: "{{.DEFAULT_CLUSTER_ADMIN_KEY}}"
    log_level: "debug"
    log_output: "stdout"
    http_listen: ":8080"
    https_listen: ":8443"
    enable_profiling: true
    kubernetes:
      namespace_selector:
      - %s
      apisix_route_version: "apisix.apache.org/v2beta2"
      watch_endpoint_slices: true
`
)
