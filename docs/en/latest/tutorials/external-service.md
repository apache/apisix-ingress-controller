---
title: Using External Services In ApisixUpstream
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

In this tutorial, we will introduce how to configure external services in the ApisixUpstream resources.

## Prerequisites

- an available Kubernetes cluster
- an available APISIX and APISIX Ingress Controller installation

We assume that your APISIX is installed in the `apisix` namespace.

## Introduction

APISIX ingress supports configuring external services as backends, both for K8s external name services and direct domains.
In this case, we don't configure the `backends` field in the ApisixRoute resource. Instead, we will use the `upstreams` field to refer to an ApisixUpstream resources with the `externalNodes` field configured.

For example:

```yaml
# httpbin-route.yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - local.httpbin.org
      paths:
      - /*
    # backends:  # We won't use the `backends` field
    #    - serviceName: httpbin
    #      servicePort: 80
    upstreams:
    - name: httpbin-upstream
```

This configuration tells the ingress controller not to resolve upstream hosts through the K8s services, but to use the configuration defined in the referenced ApisixUpstream.
The referenced ApisixUpstream *MUST* have `externalNodes` field configured. For example:

```yaml
# httpbin-upstream.yaml
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-upstream
spec:
  externalNodes:
  - type: Domain
    name: httpbin.org
```

In this yaml example, we configured `httpbin.org` as the backend. The type `Domain` indicates that this is a third-party service, and any domain name is supported here.

If you want to use an external name service in the K8s cluster, the type should be `Service` and the name should be the service name. By configuring ApisixUpstream with type `Service`, the ingress controller will automatically keep track of the content of the external name service and its changes.

## External Domain Upstream

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - local.httpbin.org
      paths:
      - /*
    upstreams:
    - name: httpbin-upstream
---
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-upstream
spec:
  externalNodes:
  - type: Domain
    name: httpbin.org
```

After applying the above configuration, we can try to access `httpbin.org` directly through APISIX.

```bash
kubectl exec -it -n apisix APISIX_POD_NAME -- curl -i -H "Host: local.httpbin.org" http://127.0.0.1:9080/get
```

If everything works, you will see the result like this:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 321
Connection: keep-alive
Date: Thu, 15 Dec 2022 10:47:30 GMT
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true
Server: APISIX/3.0.0

{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Host": "local.httpbin.org",
    "User-Agent": "curl/7.29.0",
    "X-Amzn-Trace-Id": "Root=xxxxx",
    "X-Forwarded-Host": "local.httpbin.org"
  },
  "origin": "127.0.0.1, xxxxxxxxx",
  "url": "http://local.httpbin.org/get"
}
```

The header `Server: APISIX/3.0.0` indicates that the request is sent from APISIX.

## External Name Service Upstream

Let's deploy a simple httpbin app in the namespace `test` as the backend for the external name service we will create later.

```bash
kubectl create ns test
kubectl -n test run httpbin --image-pull-policy IfNotPresent --image=kennethreitz/httpbin --port 80
kubectl -n test expose pod/httpbin --port 80
```

Then use the following configuration to create an external name service in the `apisix` namespace.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ext-httpbin
spec:
  type: ExternalName
  externalName: httpbin.test.svc
```

Now we can create an external name service ApisixRoute and ApisixUpstream.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: ext-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - ext.httpbin.org
      paths:
      - /*
    upstreams:
    - name: ext-upstream
---
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: ext-upstream
spec:
  externalNodes:
  - type: Service
    name: ext-httpbin
```

Once the configurations is synced, try to access it with the following command.

The only argument that changes is the header we pass.

```bash
kubectl exec -it -n apisix APISIX_POD_NAME -- curl -i -H "Host: ext.httpbin.org" http://127.0.0.1:9080/get
```

The output should be like:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 234
Connection: keep-alive
Date: Thu, 15 Dec 2022 10:54:21 GMT
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true
Server: APISIX/3.0.0

{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Host": "ext.httpbin.org",
    "User-Agent": "curl/7.29.0",
    "X-Forwarded-Host": "ext.httpbin.org"
  },
  "origin": "127.0.0.1",
  "url": "http://ext.httpbin.org/get"
}
```

## Domain In External Name Service

The external name service can also hold any domain name outside of the K8s cluster.

Let's update the external service configuration we applied in the previous section.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ext-httpbin
spec:
  type: ExternalName
  externalName: httpbin.org
```

Try accessing it again, and the output should contain multiple `origin`, and an `X-Amzn-Trace-Id` header, which means we are accessing the actual online `httpbin.org` service.
