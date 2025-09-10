---
title: Configuration
keywords:
  - APISIX Ingress
  - Apache APISIX
  - Kubernetes Ingress
  - Gateway API
description: Configuration of the APISIX Ingress Controller
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

The APISIX Ingress Controller is a Kubernetes Ingress Controller that implements the Gateway API. This document describes how to configure the APISIX Ingress Controller.

## Example

```yaml
log_level: "info"                               # The log level of the APISIX Ingress Controller.
                                                # the default value is "info".

controller_name: apisix.apache.org/apisix-ingress-controller  # The controller name of the APISIX Ingress Controller,
                                                          # which is used to identify the controller in the GatewayClass.
                                                          # The default value is "apisix.apache.org/apisix-ingress-controller".

leader_election_id: "apisix-ingress-controller-leader" # The leader election ID for the APISIX Ingress Controller.
                                                        # The default value is "apisix-ingress-controller-leader".
```

### Controller Name

The `controller_name` field is used to identify the `controllerName` in the GatewayClass.

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  namespace: ingress-apisix
  name: apisix
spec:
  controllerName: "apisix.apache.org/apisix-ingress-controller"
```
