---
title: Key Authentication
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
description: Explore how to configure key authentication in APISIX using APISIX Ingress Controller, which implement access control to your APIs.
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

APISIX has a flexible plugin extension system and a number of existing plugins for user authentication and authorization.

In this tutorial, you will create a Consumer, configure its key authentication Credential, and enable key authentication on a route, using APISIX Ingress Controller.

## Prerequisite

1. Complete [Get APISIX and APISIX Ingress Controller](./get-apisix-ingress-controller.md).

## Configure Key Authentication

For demonstration purpose, you will be creating a route to the [publicly hosted httpbin services](https://httpbin.org). If you would like to proxy requests to services on Kubernetes, please modify accordingly.

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

Create a Kubernetes manifest file to configure a consumer:

```yaml title="consumer.yaml"
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: tom
spec:
  gatewayRef:
    name: apisix
  credentials:
    - type: key-auth
      name: primary-key
      config:
        key: secret-key
```

Create a Kubernetes manifest file to configure a route and enable key authentication:

```yaml title="httpbin-route.yaml"
apiVersion: v1
kind: Service
metadata:
  name: httpbin-external-domain
spec:
  type: ExternalName
  externalName: httpbin.org
---
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: auth-plugin-config
spec:
  plugins:
    - name: key-auth
      config:
        _meta:
          disable: false
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
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
        name: auth-plugin-config
    backendRefs:
    - name: httpbin-external-domain
      port: 80
```

Apply the configurations to your cluster:

```shell
kubectl apply -f consumer.yaml -f httpbin-route.yaml
```

</TabItem>

<TabItem value="apisix-crd">

Create a Kubernetes manifest file to configure a consumer:

```yaml title="consumer.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: tom
spec:
  ingressClassName: apisix
  authParameter:
    keyAuth:
      value:
        key: secret-key
```

Create a Kubernetes manifest file to configure a route and enable key authentication:

```yaml title="httpbin-route.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-external-domain
spec:
  externalNodes:
  - type: Domain
    name: httpbin.org
---
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
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
      authentication:
        enable: true
        type: keyAuth
```

Apply the configurations to your cluster:

```shell
kubectl apply -f consumer.yaml -f httpbin-route.yaml
```

</TabItem>

</Tabs>

## Verify

Expose the service port to your local machine by port forwarding:

```shell
kubectl port-forward svc/apisix-gateway 9080:80 &
```

Send a request without the `apikey` header.

```shell
curl -i "http://127.0.0.1:9080/ip"
```

You should receive an an `HTTP/1.1 401 Unauthorized` response.

Send a request with a wrong key in the `apikey` header.

```shell
curl -i "http://127.0.0.1:9080/ip" -H 'apikey: wrong-key'
```

Since the key is incorrect, you should receive an `HTTP/1.1 401 Unauthorized` response.

Send a request with the correct key in the `apikey` header.

```shell
curl -i "http://127.0.0.1:9080/ip" -H 'apikey: secret-key'
```

You should receive an `HTTP/1.1 200 OK` response.
