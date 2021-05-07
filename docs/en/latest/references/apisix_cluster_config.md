---
title: ApisixRoute/v2alpha1 Reference
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

## Spec

Spec describes the desired state of an ApisixClusterConfig object.

|     Field     |  Type    |     Description       |
|---------------|----------|-----------------------|
| monitoring | object | Monitoring settings. |
| monitoring.prometheus | object | Prometheus settings. |
| monitoring.prometheus.enable | boolean | Whether to enable Prometheus or not. |
| monitoring.skywalking | object | Skywalking settings. |
| monitoring.skywalking.enable | boolean | Whether to enable Skywalking or not. |
| monitoring.skywalking.sampleRatio | number | The sample ratio for spans, value should be in `[0, 1]`.|
| admin | object | Administrative settings. |
| admin.baseURL | string | the base url for APISIX cluster. |
| admin.AdminKey | string | admin key used for authentication with APISIX cluster. |
