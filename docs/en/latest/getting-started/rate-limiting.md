---
title: Rate Limiting
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
description: Implement rate limiting in APISIX using APISIX Ingress Controller to control traffic flow, protect your APIs from misuse, and ensure fair usage by setting request limits.
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

Rate limiting is one of the commonly used techniques to protect and manage APIs. For example, you can configure your API endpoints to allow for a set number of requests within a given period of time. This ensures fair usage of the upstream services and safeguards the APIs from potential cyber attacks like DDoS (Distributed Denial of Service) or excessive requests from web crawlers.

In this tutorial, you will enable the `limit-count` plugin to set a rate limiting constraint on the incoming traffic, using APISIX Ingress Controller.

## Prerequisite

1. Complete [Get APISIX and APISIX Ingress Controller](./get-apisix-ingress-controller.md).

## Configure Rate Limiting

For demonstration purpose, you will be creating a route to the [publicly hosted httpbin services](https://httpbin.org) and [mock.api7.ai](https://mock.api7.ai). If you would like to proxy requests to services on Kubernetes, please modify accordingly.

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

</details>

If you are using Ingress or APISIX custom resources, you can proceed without additional configuration.

:::

Create a Kubernetes manifest file for a route and enable `limit-count`:

<Tabs
groupId="k8s-api"
defaultValue="gateway-api"
values={[
{label: 'Gateway API', value: 'gateway-api'},
{label: 'APISIX CRD', value: 'apisix-crd'}
]}>

<TabItem value="gateway-api">

```yaml title="httpbin-route.yaml"
apiVersion: v1
kind: Service
metadata:
  namespace: ingress-apisix
  name: httpbin-external-domain
spec:
  type: ExternalName
  externalName: httpbin.org
---
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  namespace: ingress-apisix
  name: limit-count-plugin-config
spec:
  plugins:
    - name: limit-count
      config:
        count: 2
        time_window: 10
        rejected_code: 429
---
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
    filters:
    - type: ExtensionRef
      extensionRef:
        group: apisix.apache.org
        kind: PluginConfig
        name: limit-count-plugin-config
    backendRefs:
    - name: httpbin-external-domain
      port: 80
```

</TabItem>

<TabItem value="apisix-crd">

```yaml title="httpbin-route.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  namespace: ingress-apisix
  name: httpbin-external-domain
spec:
  externalNodes:
  - type: Domain
    name: httpbin.org
---
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
      upstreams:
      - name: httpbin-external-domain
      plugins:
      - name: limit-count
        enable: true
        config:
          count: 2
          time_window: 10
          rejected_code: 429
```

</TabItem>

</Tabs>

Apply the configuration to your cluster:

```shell
kubectl apply -f httpbin-route.yaml
```

## Verify

Expose the service port to your local machine by port forwarding:

```shell
kubectl port-forward svc/apisix-gateway 9080:80 &
```

Generate 50 simultaneous requeststo the route:

```shell
resp=$(seq 50 | xargs -I{} curl "http://127.0.0.1:9080/ip" -o /dev/null -s -w "%{http_code}\n") && \
  count_200=$(echo "$resp" | grep "200" | wc -l) && \
  count_429=$(echo "$resp" | grep "429" | wc -l) && \
  echo "200": $count_200, "429": $count_429
```

The results are as expected: out of the 50 requests, 2 requests were sent successfully (status code `200`) while the others were rejected (status code `429`).

```text
"200": 2, "429": 48
```
