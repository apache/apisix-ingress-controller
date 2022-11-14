---
title: ApisixPluginConfig/v2
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixPluginConfig
description: Reference for ApisixPluginConfig/v2 custom Kubernetes resource.
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

See the [definition](https://github.com/apache/apisix-ingress-controller/blob/master/samples/deploy/crd/v1/ApisixPluginConfig.yaml) on GitHub.

| Field            | Type    | Description                                                                                                                                    |
|------------------|---------|------------------------------------------------------------------------------------------------------------------------------------------------|
| plugins          | array   | Plugins that will be executed on the Route.                                                                                                    |
| plugins[].name   | string  | Name of the Plugin. See [Plugin hub](https://apisix.apache.org/plugins/) for a list of available Plugins.                                      |
| plugins[].enable | boolean | When set to `true`, enables the Plugin.                                                                                                        |
| plugins[].config | object  | Configuration of the Plugin. Must have the same fields as in the [Plugin docs](https://apisix.apache.org/docs/apisix/plugins/batch-requests/). |
