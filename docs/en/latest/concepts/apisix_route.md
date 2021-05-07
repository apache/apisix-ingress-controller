---
title: ApisixRoute
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

`ApisixRoute` is a CRD resource which focus on how to route traffic to
expected backend, it exposes many features supported by Apache APISIX.
Compared to [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/),
functions are implemented in a more native way, with stronger semantics.

Path based route rules
----------------------

URI path are always used to split traffic, for instance, requests with host `foo.com` and
`/foo` prefix should be routed to service `foo` while requests which path is `/bar`
should be routed to service `bar`, in the manner of `ApisixRoute`, the configuration
should be:

```yaml
apiVersion: apisix.apache.org/v2alpha1
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
      - "/foo*"
    backend:
     serviceName: foo
     servicePort: 80
  - name: bar
    match:
      paths:
        - "/bar"
    backend:
      serviceName: bar
      servicePort: 80
```

There are two path types can be used, `prefix` and `exact`, default is `exact`,
while if `prefix` is desired, just append a `*`, for instance, `/id/*` matches
all paths with the prefix of `/id/`.

Advanced route features
-----------------------

Path based route are most common, but if it's not enough, try
other route features in `ApisixRoute` such as `methods`, `exprs`.

The `methods` splits traffic according to the HTTP method, the following configurations routes requests
with `GET` method to `foo` service (a Kubernetes Service).

```yaml
apiVersion: apisix.apache.org/v2alpha1
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

The `exprs` allows user to configure match conditions with arbitrary predicates in HTTP, such as queries, HTTP headers, Cookie.
It's composed by several expressions, which in turn composed by subject, operator and value/set.

```yaml
apiVersion: apisix.apache.org/v2alpha1
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
            value: 2143
      backends:
        - serviceName: foo
          servicePort: 80
```

The above configuration configures an extra route match condition, which asks the
query `id` must be equal to `2143`.

Service Resolution Granularity
------------------------------

By default a referenced Service will be watched, so
it's newest endpoints list can be updated to Apache APISIX.
apisix-ingress-controller provides another mechanism that just use
the `ClusterIP` of this service, if that's what you want, just set
the `resolveGranularity` to `service` (default is `endpoint`).

```yaml
apiVersion: apisix.apache.org/v2alpha1
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

Weight Based Traffic Split
--------------------------

There can more than one backend specified in one route rule,
when multiple backends co-exist there, the traffic split based on weights
will be applied (which actually uses the [traffic-split](http://apisix.apache.org/docs/apisix/plugins/traffic-split/) plugin in Apache APISIX).
You can specify weight for each backend, the default weight is `100`.

```yaml
apiVersion: apisix.apache.org/v2alpha1
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

The above `ApisixRoute` has one route rule, which contains two backends `foo` and `bar`, the weight ratio is `100:50`,
which means `2/3` requests (with `GET` method and `User-Agent` matching regex pattern `.*Chrome.*`) will be sent to service `foo` and `1/3` requests
will be proxied to serivce `bar`.

Plugins
-------

Apache APISIX provides more than 40 [plugins](https://github.com/apache/apisix/tree/master/docs/en/latest/plugins), which can be used
in `ApisixRoute`. All configuration items are named same to the one in APISIX.

```yaml
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
    - name: httpbin
      match:
        hosts:
        - local.httpbin.org
        paths:
          - /*
      backends:
        - serviceName: foo
          servicePort: 80
      plugins:
        - name: cors
          enable: true
```

The above configuration enables [Cors](https://github.com/apache/apisix/blob/master/docs/en/latest/plugins/cors.md) plugin for requests
which host is `local.httpbin.org`.

Websocket Proxy
---------------

[Websocket](https://en.wikipedia.org/wiki/WebSocket#:~:text=WebSocket%20is%20a%20computer%20communications,WebSocket%20is%20distinct%20from%20HTTP.) service can be proxied
by creating a route with specifying the `websocket` field.

```yaml
apiVersion: apisix.apache.org/v2alpha1
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

TCP Route
---------

apisix-ingress-controller supports the port-based tcp route.

```yaml
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: tcp-route
spec:
  tcp:
    - name: tcp-route-rule1
      match:
        ingressPort: 9100
      backend:
        serviceName: tcp-server
        servicePort: 8080
```

The above yaml configuration guides TCP traffic entered to the Ingress proxy server (i.e. [APISIX](https://apisix.apache.org)) port `9100` should be routed to the backend service `tcp-server`.

Note since APISIX doesn't support dynamic listening, so here the `9100` port should be pre-defined in APISIX [configuration](https://github.com/apache/apisix/blob/master/conf/config-default.yaml#L101).
