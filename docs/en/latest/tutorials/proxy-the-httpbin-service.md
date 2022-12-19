---
title: Configuring Ingress with APISIX CRDs
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes Ingress
  - APISIX CRDs
description: A tutorial on configuring Ingress using APISIX Custom Resource Definitions (CRDs).
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

This tutorial walks through configuring APISIX Ingress with [APISIX Custom Resource Definitions (CRDs)](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_route).

## Prerequisites

Before you move on, make sure you:

1. Have access to a Kubernetes cluster. This tutorial uses [minikube](https://github.com/kubernetes/minikube).
2. Install APISIX Ingress. See the [Installation](https://apisix.apache.org/docs/ingress-controller/deployments/minikube) section.

## Deploy httpbin

We will deploy a sample service, [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/), for this tutorial.

You can deploy it to your Kubernetes cluster by running:

```shell
kubectl run httpbin --image kennethreitz/httpbin --port 80
kubectl expose pod httpbin --port 80
```

## Configuring Ingress

We can configure the Ingress using an [ApisixRoute](https://apisix.apache.org/docs/ingress-controller/references/apisix_route_v2) resource. The example below configures APISIX to route requests to the httpbin service:

```yaml title="httpbin-ingress.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpserver-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - local.httpbin.org
      paths:
      - /*
    backends:
       - serviceName: httpbin
         servicePort: 80
```

This configuration will route all requests with host `local.httpbin.org` to the httpbin service.

You can apply it by running:

```shell
kubectl apply -f httpbin-ingress.yaml
```

## Test the created Routes

If you followed along and used minikube and `NodePort` service to expose APISIX, you can access it through the Node IP of the service `apisix-gateway`. If the Node IP is not reachable directly (if you are on Darwin, Windows, or WSL), you can create a tunnel to access the service on your machine:

```shell
minikube service apisix-gateway --url -n ingress-apisix
```

Now, you can send a `GET` request to the created Route and it will be Routed to the httpbin service:

```shell
curl --location --request GET "localhost:57687/get?foo1=bar1&foo2=bar2" -H "Host: local.httpbin.org"
```

You will receive a response similar to:

```json title="output"
{
  "args": {
    "foo1": "bar1", 
    "foo2": "bar2"
  }, 
  "headers": {
    "Accept": "*/*", 
    "Host": "local.httpbin.org", 
    "User-Agent": "curl/7.84.0", 
    "X-Forwarded-Host": "local.httpbin.org"
  }, 
  "origin": "172.17.0.1", 
  "url": "http://local.httpbin.org/get?foo1=bar1&foo2=bar2"
}
```
