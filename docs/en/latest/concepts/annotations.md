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

## Customized http methods

> `http-allow-methods` and `http-block-methods` are mutually exclusive.
> If they're both set, only `http-allow-methods` works

### Allow http methods

This annotation can be used to specify which http method are allowed. Multiple methods can also be specified by separating them with commas.

```yaml
k8s.apisix.apache.org/http-allow-methods: "GET,POST"
```

The default value is empty which means all methods are allowed.

### Block http methods

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
    k8s.apisix.apache.org/rewrite-target-regex: "/app/(.*)"
    k8s.apisix.apache.org/rewrite-target-regex-template: "/$1"
  name: ingress-v1
spec:
  ingressClassName: apisix
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
    k8s.apisix.apache.org/http-to-https: "true"
  name: ingress-v1
spec:
  ingressClassName: apisix
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
    k8s.apisix.apache.org/use-regex: "true"
  name: ingress-v1
spec:
  ingressClassName: apisix
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
    k8s.apisix.apache.org/enable-websocket: "true"
  name: ingress-v1
spec:
  ingressClassName: apisix
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

## Response Rewrite

You can enable [Response Rewrite](https://github.com/apache/apisix/blob/master/docs/en/latest/plugins/response-rewrite.md) by adding the annotation as shown below:

```yaml
metadata:
  annotations:
    k8s.apisix.apache.org/enable-response-rewrite: "true"
```

You can customize the behaviour with some additional annotations as shown below.

### New HTTP status code

This annotation configures the new HTTP status code in the response.

```yaml
k8s.apisix.apache.org/response-rewrite-status-code: "404"
```

If unset, falls back to the original status code.

### New body

This annotation configures the new body in the response.

```yaml
k8s.apisix.apache.org/response-rewrite-body: "bar-body"
```

### Add header

This annotation configures to append the new headers in the response.

```yaml
k8s.apisix.apache.org/response-rewrite-add-header: "testkey1:testval1,testkey2:testval2"
```

### Set header

This annotation configures to rewrite the new headers in the response.

```yaml
k8s.apisix.apache.org/response-rewrite-set-header: "testkey1:testval1,testkey2:testval2"
```

### Remove header

This annotation configures to remove headers in the response.

```yaml
k8s.apisix.apache.org/response-rewrite-remove-header: "testkey1,testkey2"
```

### Body Base64

When set, the body of the request will be decoded before writing to the client.

```yaml
k8s.apisix.apache.org/response-rewrite-body-base64: "true"
```

The default value is `"false"`.

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
    k8s.apisix.apache.org/plugin-config-name: "echo-and-cors-apc"
  name: ingress-v1
spec:
  ingressClassName: apisix
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

## Upstream scheme

The scheme used when communicating with the Upstream. this value can be one of 'http', 'https', 'grpc', 'grpcs'. Defaults to 'http'.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/upstream-scheme: grpcs
  name: ingress-v1
spec:
  rules:
  - host: e2e.apisix.local
    http:
      paths:
      - path: /helloworld.Greeter/SayHello
        pathType: ImplementationSpecific
        backend:
          service:
            name: test-backend-service-e2e-test
            port:
              number: 50053
```

## Cross-namespace references

This annotation can be used to route to services in a different namespace.

In the example configuration below, the Ingress resource in the `default` namespace references the httpbin service in the `test` namespace:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    k8s.apisix.apache.org/svc-namespace: test
  name: ingress-v1-svc
  namespace: default
spec:
  ingressClassName: apisix
  rules:
    - host: httpbin.org
      http:
        paths:
          - path: /ip
            pathType: Exact
            backend:
              service:
                name: httpbin
                port:
                  number: 80
```

## Upstream retries

This annotation can be used to configure retries among multiple nodes in an upstream. You may want the proxy to retry when requests occur faults like transient network errors or service unavailable, By default the retry count is 1. You can change it by specifying the retries field.

The following configuration configures the retries to 3, which indicates there'll be at most 3 requests sent to Kubernetes service httpbin's endpoints.

One should bear in mind that passing a request to the next endpoint is only possible if nothing has been sent to a client yet. That is, if an error or timeout occurs in the middle of the transferring of a response, fixing this is impossible.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    k8s.apisix.apache.org/upstream-retries: "3"
  name: ingress-ext-v1beta1
spec:
  ingressClassName: apisix
  rules:
    - host: httpbin.org
      http:
        paths:
          - path: /ip
            pathType: Exact
            backend:
              service:
                name: httpbin
                port:
                  number: 80
```

## Upstream timeout

This annotation can be used to configure different types of timeout on an upstream. The default connect, read and send timeout are 60s, which might not be proper for some applications.

The below example sets the read, connect and send timeout to 5s, 10s, 10s respectively.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    k8s.apisix.apache.org/upstream-read-timeout.: "5s"
    k8s.apisix.apache.org/upstream-connect-timeout: "10s"
    k8s.apisix.apache.org/upstream-send-timeout: "10s"
  name: ingress-ext-v1beta1
spec:
  ingressClassName: apisix
  rules:
    - host: httpbin.org
      http:
        paths:
          - path: /ip
            pathType: Exact
            backend:
              service:
                name: httpbin
                port:
                  number: 80
```
