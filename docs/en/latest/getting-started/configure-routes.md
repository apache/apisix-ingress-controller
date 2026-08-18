---
title: Configure Routes
keywords:
  - APISIX Ingress Controller
  - ApisixRoute
  - Kubernetes Gateway API
  - Kubernetes Ingress
description: Configure routes to Kubernetes Services with Gateway API, Kubernetes Ingress, or the ApisixRoute CRD using APISIX Ingress Controller.
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

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

APISIX Ingress Controller translates Kubernetes routing resources into Apache APISIX configuration. You can use Gateway API, Kubernetes Ingress, or the APISIX-native `ApisixRoute` custom resource to define request matching and target services.

This tutorial creates the same HTTP route with each API and verifies how APISIX proxies traffic to an httpbin Service. See [APISIX Ingress Controller Resources](../concepts/resources.md) for a comparison of the supported resource types and the [ApisixRoute API reference](../reference/api-reference.md#apisixroute) for field-level details.

## Prerequisites

1. Complete [Get APISIX and APISIX Ingress Controller](./get-apisix-ingress-controller.md).

## Set Up a Sample Upstream

Install the httpbin example application on the cluster to test the configuration:

```bash
kubectl apply -f https://raw.githubusercontent.com/apache/apisix-ingress-controller/refs/heads/v2.1.0/examples/httpbin/deployment.yaml
```

## Configure a Route

In this section, you will create a Route that forwards client requests to the httpbin example application, an HTTP request and response service.

Choose the resource that matches how your Kubernetes platform manages traffic:

- Gateway API `HTTPRoute` provides a portable Kubernetes routing API.
- Kubernetes `Ingress` supports the standard Ingress API.
- `ApisixRoute` exposes APISIX-specific routing capabilities through a custom resource.

:::important

If you are using Gateway API, you should first configure the GatewayClass and Gateway resources:

<details>

<summary>Show configuration</summary>

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  namespace: ingress-apisix
  name: apisix
spec:
  controllerName: apisix.apache.org/apisix-ingress-controller
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  namespace: ingress-apisix
  name: apisix
spec:
  gatewayClassName: apisix
  listeners:
  - name: http
    protocol: HTTP
    port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-config
```

The `port` in the Gateway listener can be used for route matching based on [`listener_port_match_mode`](../reference/configuration-file.md) (`off` by default; `auto` or `explicit` opt in). The controller cannot dynamically open new ports on the data plane, so ensure APISIX is configured to listen on the port.

</details>

If you are using Ingress or APISIX custom resources, you can proceed without additional configuration, as the IngressClass resource below is already applied with installation:

<details>

<summary>Show configuration</summary>

```yaml
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: apisix.apache.org/apisix-ingress-controller
  parameters:
    apiGroup: apisix.apache.org
    kind: GatewayProxy
    name: apisix-config
    namespace: ingress-apisix
    scope: Namespace
```

</details>

See [Define Controller and Gateway](../reference/example.md#define-controller-and-gateway) for more information on parameters.

:::

Create a Kubernetes manifest file for a Route that proxy requests to httpbin:

<Tabs
groupId="k8s-api"
defaultValue="gateway-api"
values={[
{label: 'Gateway API', value: 'gateway-api'},
{label: 'Ingress', value: 'ingress-rs'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway-api">

```yaml title="httpbin-route.yaml"
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  namespace: ingress-apisix
  name: getting-started-ip
spec:
  parentRefs:
  - name: apisix
  rules:
  - matches:
    - path:
        type: Exact
        value: /ip
    backendRefs:
    - name: httpbin
      port: 80
```

</TabItem>

<TabItem value="ingress-rs">

```yaml title="httpbin-route.yaml"
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  namespace: ingress-apisix
  name: getting-started-ip
spec:
  ingressClassName: apisix
  rules:
    - http:
        paths:
          - backend:
              service:
                name: httpbin
                port:
                  number: 80
            path: /ip
            pathType: Exact
```

</TabItem>

<TabItem value="apisix-crd">

Use `ApisixRoute` when you need APISIX-native route configuration. For all available fields, see the [ApisixRoute API reference](../reference/api-reference.md#apisixroute).

```yaml title="httpbin-route.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  namespace: ingress-apisix
  name: getting-started-ip
spec:
  ingressClassName: apisix
  http:
    - name: getting-started-ip
      match:
        paths:
          - /ip
      backends:
        - serviceName: httpbin
          servicePort: 80
```

</TabItem>

</Tabs>

Apply the configurations to your cluster:

```shell
kubectl apply -f httpbin-route.yaml
```

## Verify

Expose the service port to your local machine by port forwarding:

```shell
kubectl port-forward svc/apisix-gateway 9080:80 &
```

Send a request to the Route:

```shell
curl "http://127.0.0.1:9080/ip"
```

You should see a response similar to the following:

```json
{
  "origin": "127.0.0.1"
}
```
