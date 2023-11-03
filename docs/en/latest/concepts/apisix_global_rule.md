---
title: ApisixGlobalRule
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixGlobalRule
description: Guide to using ApisixGlobalRule custom Kubernetes resource.
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

`ApisixGlobalRule` is a Kubernetes CRD resource used to create an APISIX [global-rule](https://apisix.apache.org/docs/apisix/terminology/global-rule/) object, which can apply the [plugin](https://apisix.apache.org/docs/apisix/next/terminology/plugin/) to all requests.

## Example

Enable the [limit-count](https://apisix.apache.org/docs/apisix/next/plugins/limit-count/) plugin on the APISIX, which can limit all requests.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: global
spec:
  plugins:
  - name: limit-count
    enable: true 
    config:
      time_window": 60,
      policy: "local",
      count: 2,
      key: "remote_addr",
      rejected_code: 503
```
