---
title: ApisixRoute/v1 (Deprecated) Reference
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

**WARNINIG**: `ApisixRoute/v1` is obsolete and will be unsupported in the future, please use `ApisixRoute/v2alpha1`!

|     Field     |  Type    |                    Description                     |
|---------------|----------|----------------------------------------------------|
| rules         | array    | ApisixRoute's request matching rules.              |
| host          | string   | The requested host.                                |
| http          | object   | Route rules are applied to the scope of layer 7 traffic.     |
| paths         | array    | Path-based `route` rule matching.                     |
| backend       | object   | Backend service information configuration.         |
| serviceName   | string   | The name of backend service. `namespace + serviceName + servicePort` form an unique identifier to match the back-end service.                      |
| servicePort   | int      | The port of backend service. `namespace + serviceName + servicePort` form an unique identifier to match the back-end service.                      |
| path          | string   | The URI matched by the route. Supports exact match and prefix match. Example，exact match: `/hello`, prefix match: `/hello*`.                     |
| plugins       | array    | Custom plugin collection (Plugins defined in the `route` level). For more plugin information, please refer to the [Apache APISIX plugin docs](https://github.com/apache/apisix/tree/master/docs/en/latest/plugins).    |
| name          | string   | The name of the plugin. For more information about the example plugin, please check the [limit-count docs](https://github.com/apache/apisix/blob/master/docs/en/latest/plugins/limit-count.md#Attributes).             |
| enable        | boolean  | Whether to enable the plugin, `true`: means enable, `false`: means disable.      |
| config        | object   | Configuration of plugin information. Note: The check of configuration schema is missing now, so please be careful when editing.    |
