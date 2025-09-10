---
title: Configure Routes
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
description: Learn how to create routes in APISIX using APISIX Ingress controller to forward client to upstream services.
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

Apache APISIX provides flexible gateway management capabilities based on routes, in which routing paths and target upstreams are defined.

This tutorial guides you through creating a Route using the APISIX Ingress Controller and verifying its behavior. You’ll configure a Route to a sample Upstream pointing to an httpbin service, then send a request to observe how APISIX proxies the traffic.

## Prerequisites

1. Complete [Get APISIX and APISIX Ingress Controller](./get-apisix-ingress-controller.md).

## Set Up a Sample Upstream

Install the httpbin example application on the cluster to test the configuration:

```bash
kubectl apply -f https://raw.githubusercontent.com/apache/apisix-ingress-controller/refs/heads/v2.0.0/examples/httpbin/deployment.yaml
```

## Configure a Route

In this section, you will create a Route that forwards client requests to the httpbin example application, an HTTP request and response service.

You can use either Gateway API, Ingress, or APISIX CRD resources to configure the route.

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
