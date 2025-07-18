---
title: Load Balancing
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
description: Learn how to implement load balancing in APISIX using APISIX Ingress Controller, distributing clients requests across multiple upstream nodes.
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

Load balancing is a technique used to distribute network request loads. It is a key consideration in designing systems that need to handle a large volume of traffic, allowing for improved system performance, scalability, and reliability.

In this tutorial, you will create a route using APISIX Ingress Controller with two upstream services and uses the round-robin load balancing algorithm to load balance requests.

## Prerequisite

1. Complete [Get APISIX and APISIX Ingress Controller](./get-apisix-ingress-controller.md).

## Configure Load Balancing

For demonstration purpose, you will be creating a route to the [publicly hosted httpbin services](https://httpbin.org) and [mock.api7.ai](https://mock.api7.ai). If you would like to proxy requests to services on Kubernetes, please modify accordingly.

:::important

If you are using Gateway API, you should first configure the GatewayClass and Gateway resources:

<details>

<summary>Show configuration</summary>

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: apisix
spec:
  controllerName: apisix.apache.org/apisix-ingress-controller
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
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

</details>

If you are using Ingress or APISIX custom resources, you can proceed without additional configuration.

:::

<Tabs
groupId="k8s-api"
defaultValue="gateway-api"
values={[
{label: 'Gateway API', value: 'gateway-api'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway-api">

APISIX Ingress controller installed with the current helm chart version (`apisix-2.11.2`) has a bug in load balancing, which is actively being fixed.

</TabItem>

<TabItem value="apisix-crd">

Create a Kubernetes manifest file for a route that proxy requests to two upstream services for load balancing:

```yaml title="lb-route.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-external-domain
spec:
  scheme: https
  passHost: node
  externalNodes:
  - type: Domain
    name: httpbin.org
    weight: 1
    port: 443
---
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: mockapi7-external-domain
spec:
  scheme: https
  passHost: node
  externalNodes:
  - type: Domain
    name: mock.api7.ai
    weight: 1
    port: 443
---
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: lb-route
spec:
  ingressClassName: apisix
  http:
    - name: lb-route
      match:
        paths:
          - /headers
      upstreams:
      - name: httpbin-external-domain
      - name: mockapi7-external-domain
```

Apply the configuration to your cluster:

```shell
kubectl apply -f lb-route.yaml
```

</TabItem>

</Tabs>

## Verify

Expose the service port to your local machine by port forwarding:

```shell
kubectl port-forward svc/apisix-gateway 9080:80 &
```

Generate 50 consecutive requests to the route to see the load-balancing effect:

```shell
resp=$(seq 50 | xargs -I{} curl "http://127.0.0.1:9080/headers" -sL) && \
  count_httpbin=$(echo "$resp" | grep "httpbin.org" | wc -l) && \
  count_mockapi7=$(echo "$resp" | grep "mock.api7.ai" | wc -l) && \
  echo httpbin.org: $count_httpbin, mock.api7.ai: $count_mockapi7
```

The command keeps count of the number of requests that was handled by the two services respectively. The output shows that requests were distributed over to the two services:

```text
httpbin.org: 23, mock.api7.ai: 27
```

The distribution of requests across services should be close to 1:1 but might not always result in a perfect 1:1 ratio. The slight deviation is due to APISIX operates with multiple workers.
