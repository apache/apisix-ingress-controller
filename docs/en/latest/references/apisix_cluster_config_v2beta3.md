---
title: ApisixClusterConfig/v2beta3
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixClusterConfig
description: Reference for ApisixClusterConfig/v2beta3 custom Kubernetes resource.
---
<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

See [concepts](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_cluster_config) to learn more about how to use the ApisixClusterConfig resource.

## Spec

See the [definition](https://github.com/apache/apisix-ingress-controller/blob/master/samples/deploy/crd/v1/ApisixClusterConfig.yaml) on GitHub.

| Attribute                         | Type    | Description                                    |
|-----------------------------------|---------|------------------------------------------------|
| monitoring                        | object  | Monitoring configurations.                     |
| monitoring.prometheus             | object  | Prometheus configurations.                     |
| monitoring.prometheus.enable      | boolean | When set to `true`, enables Prometheus.        |
| monitoring.skywalking             | object  | Apache SkyWalking configurations.              |
| monitoring.skywalking.enable      | boolean | When set to `true`, enables SkyWalking.        |
| monitoring.skywalking.sampleRatio | number  | Sample ratio for spans. Should be in `[0, 1]`. |
| admin                             | object  | Admin configurations.                          |
| admin.baseURL                     | string  | Base URL of the APISIX cluster.                |
| admin.AdminKey                    | string  | Admin key to authenticate with APISIX cluster. |
