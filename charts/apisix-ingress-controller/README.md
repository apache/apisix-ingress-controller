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

# Apache APISIX Ingress Controller Helm Chart

## Prerequisites

- Kubernetes 1.12+
- [Apache APISIX](https://github.com/apache/apisix#configure-and-installation)
- [Helm v3.0+](https://helm.sh/docs/intro/quickstart/#install-helm)

## Install

To install the chart with release name `apisix-ingress-controller`:

```bash
helm install apisix-ingress-controller --namespace ingress-apisix .
```

## Uninstall

To uninstall/delete the `apisix-ingress-controller` release:

```bash
helm uninstall apisix-ingress-controller --namespace ingress-apisix
```
