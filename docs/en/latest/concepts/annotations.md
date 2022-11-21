---
title: Annotations
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
  - Annotations
description: Guide to using additional features of APISIX Ingress with annotations.
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

Annotations can be used with the [native Kubernetes ingress resource](https://kubernetes.io/docs/concepts/services-networking/ingress/) to access advanced features in Apache APISIX. Alternatively, you can use [APISIX's CRDs](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_route) for a better experience.

This document describes all the available annotations and their uses.

:::note

Key-value pairs in annotations are strings. So boolean values should be reprsented as `"true"` and `"false"`.

:::

## CORS

You can enable [CORS](https://en.wikipedia.org/wiki/Cross-origin_resource_sharing) by adding the annotation as shown below:

```yaml
metadata:
  annotations:
    k8s.apisix.apache.org/enable-cors: "true"
```

You can also customize the behaviour with some additional annotations as shown below.

### Allow origins

This annotation configures which origins are allowed. Multiple origins can be added in a comma separated form.

```yaml
k8s.apisix.apache.org/cors-allow-origin: "https://foo.com,http://bar.com:8080"
```

The default value is `"*"` which means all origins are allowed.

### Allow headers

This annotation configures which headers are allowed. Multiple headers can be added in a comma separated form.

```yaml
k8s.apisix.apache.org/cors-allow-headers: "Host: https://bar.com:8080"
```

The default value is `"*"` which means all headers are allowed.

### Allow methods

This annotation configures which HTTP methods are allowed. Multiple methods can be added in a comma separated form.

```yaml
k8s.apisix.apache.org/cors-allow-methods: "GET,POST"
```

The default value is `"*"` which means all methods are allowed.

## Allowlist source range

This annotation can be used to specify which client IP addresses or nets are allowed. Multiple IP addresses can also be specified by separating them with commas.

```yaml
k8s.apisix.apache.org/allowlist-source-range: "10.0.5.0/16,127.0.0.1,192.168.3.98"
```

The default value is empty which means all IP addresses are allowed.

## Blocklist source range

This annotation can be used to specify which client IP addresses or nets are blocked. Multiple IP addresses can also be specified by separating them with commas.

```yaml
k8s.apisix.apache.org/blocklist-source-range: "127.0.0.1,172.17.0.0/16"
```

The default value is empty which means no IP addresses are blocked.

## Allow http method

This annotation can be used to specify which http method are allowed. Multiple methods can also be specified by separating them with commas.

```yaml
k8s.apisix.apache.org/http-allow-method: "GET,POST"
```

The default value is empty which means all methods are allowed.

## Block http method

> `http-block-method` would be ignored while `http-allow-method` is also set.

This annotation can be used to specify which http method are blocked. Multiple methods can also be specified by separating them with commas.

```yaml
k8s.apisix.apache.org/http-block-method: "PUT,DELETE"
```

The default value is empty which means no methods are blocked.

## Rewrite target

These annotations are used to rewrite requests.

The annotation `k8s.apisix.apache.org/rewrite-target` specifies where to forward the request.

If you want to use regex and match groups, use the annotations `k8s.apisix.apache.org/rewrite-target-regex` and `k8s.apisix.apache.org/rewrite-target-regex-template`. The former should contain the matching rule and the latter should contain the rewrite rule. Both these annotations must be used together.

The example below configures the Ingress to forward all requests with `/app` prefix to the backend removing the `/app/` part. So, a request to `/app/ip` will be forwarded to `/ip`.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/rewrite-target-regex: "/app/(.*)"
    k8s.apisix.apache.org/rewrite-target-regex-template: "/$1"
  name: ingress-v1
spec:
  rules:
    - host: httpbin.org
      http:
        paths:
          - path: /app
            pathType: Prefix
            backend:
              service:
                name: httpbin
                port:
                  number: 80
```

## HTTP to HTTPS

This annotation is used to redirect HTTP requests to HTTPS with a `301` status code and with the same URI as the original request.

The example below will redirect HTTP requests with a `301` status code with a response header `Location:https://httpbin.org/sample`.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/http-to-https: "true"
  name: ingress-v1
spec:
  rules:
    - host: httpbin.org
      http:
        paths:
          - path: /sample
            pathType: Exact
            backend:
              service:
                name: httpbin
                port:
                  number: 80
```

## Regex paths

This annotation is can be used to enable regular expressions in path matching.

With this annotation set to `"true"` and `PathType` set to `ImplementationSpecific`, the path matching will use regex. The example below configures Ingress to route requests to path `/api/*/action1` to `service1` and `/api/*/action2` to `service2`.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/use-regex: "true"
  name: ingress-v1
spec:
  rules:
    - host: httpbin.org
      http:
        paths:
          - path: /api/.*/action1
            pathType: ImplementationSpecific
            backend:
              service:
                name: service1
                port:
                  number: 80
          - path: /api/.*/action2
            pathType: ImplementationSpecific
            backend:
              service:
                name: service2
                port:
                  number: 80
```

## Enable websocket

This annotation is use to enable websocket connections.

In the example below, the annotation will enable websocket connections on the route `/api/*`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/enable-websocket: "true"
  name: ingress-v1
spec:
  rules:
    - host: httpbin.org
      http:
        paths:
          - path: /api/*
            pathType: ImplementationSpecific
            backend:
              service:
                name: service1
                port:
                  number: 80
```

## Using ApisixPluginConfig resource

This annotation is used to enable a defined [ApisixPluginConfig](https://apisix.apache.org/docs/ingress-controller/references/apisix_pluginconfig_v2) resource on a particular route.

The value of the annotation should be the name of the created `ApisixPluginConfig` resource.

The example below shows how this is configured. The created route `/api/*` will have the [echo](https://apisix.apache.org/docs/apisix/plugins/echo/) and [cors](https://apisix.apache.org/docs/apisix/plugins/cors/) Plugins enabled as has the resource configured through annotations:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: echo-and-cors-apc
spec:
  plugins:
    - name: echo
      enable: true
      config:
        before_body: "This is the preface"
        after_body: "This is the epilogue"
        headers:
          X-Foo: v1
          X-Foo2: v2
    - name: cors
      enable: true
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/plugin-config-name: "echo-and-cors-apc"
  name: ingress-v1
spec:
  rules:
    - host: httpbin.org
      http:
        paths:
          - path: /api/*
            pathType: ImplementationSpecific
            backend:
              service:
                name: service1
                port:
                  number: 80
```
