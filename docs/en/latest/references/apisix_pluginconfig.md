---
title: ApisixPluginConfig/v2beta3 Reference
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

Spec describes the desired state of an ApisixPluginConfig object.

|     Field     |  Type    |                    Description                     |
|---------------|----------|----------------------------------------------------|
| plugins         | array    | A series of custom plugins that will be executed once this route rule is matched |
| plugins[].name | string | The plugin name, see [docs](http://apisix.apache.org/docs/apisix/getting-started) for learning the available plugins. |
| plugins[].enable | boolean | Whether the plugin is in use |
| plugins[].config | object | The plugin configuration, fields should be same as in APISIX. |
