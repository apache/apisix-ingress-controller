---
title: ApisixRoute
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixRoute
description: Guide to using ApisixRoute custom Kubernetes resource.
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

`ApisixRoute` is a Kubernetes CRD object that provides a spec to route traffic to services with APISIX. It is much more capable and easy to use compared to the default [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resource.

See [reference](https://apisix.apache.org/docs/ingress-controller/references/apisix_route_v2) for the full API documentation.

## Path-based routing

The example below shows how you can configure Ingress to route traffic to two backend services. Requests with host `foo.com` and `/foo` prefix are routed to the `foo` service and requests with the `/bar` prefix are routed to the `bar` service.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: foo-bar-route
spec:
  http:
  - name: foo
    match:
      hosts:
      - foo.com
      paths:
      - "/foo/*"
    backends:
    - serviceName: foo
      servicePort: 80
  - name: bar
    match:
      paths:
        - "/bar/*"
    backends:
    - serviceName: bar
      servicePort: 80
```

:::info IMPORTANT

Paths are matched exactly by default. To match a prefix, use `*`. For example `/id/*` will match all paths with the `/id/` prefix.

:::

## Advanced routing

`ApisixRoute` resource can also be used to configure advanced routing through `methods` and `exprs`.

The `methods` attribute can be used to route traffic based on the HTTP method as shown in the example below. This will route all requests with the `GET` method to the `foo` service.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: method-route
spec:
  http:
    - name: method
      match:
        paths:
        - /
        methods:
        - GET
      backends:
      - serviceName: foo
        servicePort: 80
```

The `exprs` attribute is used to configure conditions to match HTTP queries, headers, and cookies.

It can be composed of several expressions and each of them in-turn is composed of a subject, operator, and a value/set.

The configuration below will route all requests with a query parameter `id` with the value `2143` to the `foo` service:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: method-route
spec:
  http:
    - name: method
      match:
        paths:
          - /
        exprs:
          - subject:
              scope: Query
              name: id
            op: Equal
            value: "2143"
      backends:
      - serviceName: foo
        servicePort: 80
```

## Service resolution granularity

By default, the service referenced will be watched to update its endpoint list in APISIX. To just use the `ClusterIP` of the service, you can set the `resolveGranularity` attribute to `service` (defaults to `endpoint`):

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: method-route
spec:
  http:
    - name: method
      match:
        paths:
          - /*
        methods:
          - GET
      backends:
      - serviceName: foo
        servicePort: 80
        resolveGranularity: service
```

## Weight-based traffic split

You can configure more than one backend services in a route rule and set weights to route traffic between them. This uses the [traffic-split](http://apisix.apache.org/docs/apisix/plugins/traffic-split/) Plugin internally. The default weight is `100`.

The example below shows routing traffic between two services with a weight ratio `100:50`. This means that 2/3 of the requests (with `GET` method and `User-Agent` header matching the regex pattern `.*Chrome.*`) will be routed to the `foo` service and 1/3 of the requests will be routed to the `bar` service:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: method-route
spec:
  http:
    - name: method
      match:
        paths:
          - /*
        methods:
          - GET
        exprs:
          - subject:
              scope: Header
              name: User-Agent
            op: RegexMatch
            value: ".*Chrome.*"
      backends:
      - serviceName: foo
        servicePort: 80
        weight: 100
      - serviceName: bar
        servicePort: 81
        weight: 50
```

## Plugins

APISIX's [80+ Plugins](https://apisix.apache.org/docs/apisix/plugins/batch-requests/) can be used with APISIX Ingress. These Plugins have the same name as in the APISIX documentation.

:::note

If the Plugin is not enabled in APISIX by default, you can enable it by adding it to the `plugins` attribute in your `values.yaml` file while installing APISIX and Ingress controller via Helm. Alternatively, you can directly modify your APISIX configuration file (`conf/config.yaml`) to enable/disable Plugins.

:::

The example below configures [limit-count](https://apisix.apache.org/docs/apisix/plugins/limit-count) Plugin for the route:

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
     - httpbin.org
     paths:
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: limit-count
     enable: true
     config:
       rejected_code: 503
       count: 2
       time_window: 3
       key: remote_addr
```

You can also use the [ApisixPluginConfig](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_plugin_config) CRD to extract and reuse commonly used Plugins and bind them directly to a Route.

### Config with secretRef

Plugins are supported to be configured from kubernetes secret with `secretRef`.

The priority is `plugins.secretRef > plugins.config`. That is, the duplicated key in `plugins.config` are replaced by `plugins.secretRef`.

Example below configures echo plugin. The final values of `before_body`, `body` and `after_body` are "This is the replaced preface", "my custom body" and "This is the epilogue", respectively.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: echo
data:
  # content is "This is the replaced preface"
  before_body: IlRoaXMgaXMgdGhlIHJlcGxhY2VkIHByZWZhY2Ui
  # content is "my custom body"
  body: Im15IGN1c3RvbSBib2R5Ig==
---
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
    - name: rule1
      match:
        hosts:
          - httpbin.org
        paths:
          - /ip
      backends:
        - serviceName: %s
          servicePort: %d
          weight: 10
      plugins:
        - name: echo
          enable: true
          config:
            before_body: "This is the preface"
            after_body: "This is the epilogue"
            headers:
              X-Foo: v1
              X-Foo2: v2
          secretRef: echo
```

## Config with secretRef where the secret data contains path to a specific key that needs to be overridden in plugin config

You can also configure specific fields in the plugin configuration that are deeply nested by passing the path to that field. The path is dot-separated keys that lead to that field. The below example overrides the `X-Foo` header field in the plugin configuration from `v1` to `v2`.

```yaml
apiVersion: v1
kind: Secret
metadata:
 #content is "v2"
 name: echo
data:
 headers.X-Foo: djI=
---
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: echo
     enable: true
     config:
       before_body: "This is the preface"
       after_body: "This is the epilogue"
       headers:
         X-Foo: v1
     secretRef: echo
```

## Websocket proxy

You can route requests to [WebSocket](https://en.wikipedia.org/wiki/WebSocket#:~:text=WebSocket%20is%20a%20computer%20communications,WebSocket%20is%20distinct%20from%20HTTP.) services by setting the `websocket` attribute to `true` as shown below:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: ws-route
spec:
  http:
    - name: websocket
      match:
        hosts:
          - ws.foo.org
        paths:
          - /*
      backends:
      - serviceName: websocket-server
        servicePort: 8080
      websocket: true
```

## TCP route

You can configure APISIX Ingress to route traffic to TCP servers.

The example below configures APISIX Ingress to route traffic from port `9100` to the service `tcp-server`:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: tcp-route
spec:
  stream:
    - name: tcp-route-rule1
      protocol: TCP
      match:
        ingressPort: 9100
      backend:
        serviceName: tcp-server
        servicePort: 8080
```

:::note

The `ingressPort` (`9100` here) should be pre-defined in the [APISIX configuration](https://github.com/apache/apisix/blob/master/conf/config-default.yaml#L101).

:::

## UDP route

You can configure APISIX Ingress to route traffic to UDP servers.

The example below configures APISIX Ingress to route traffic from port `9200` to the service `udp-server`:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: udp-route
spec:
  stream:
    - name: udp-route-rule1
      protocol: UDP
      match:
        ingressPort: 9200
      backend:
        serviceName: udp-server
        servicePort: 53
```

:::note

The `ingressPort` (`9200` here) should be pre-defined in the [APISIX configuration](https://github.com/apache/apisix/blob/master/conf/config-default.yaml#L105).

:::
