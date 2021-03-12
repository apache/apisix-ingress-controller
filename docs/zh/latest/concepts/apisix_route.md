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

`ApisixRoute`是一个CRD资源，它关注如何将流量路由到指定的后端，它暴露了 Apache APISIX 支持的许多特性。
与[Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) 相比, 功能以一种更自然的方式实现，具有更强的语义。

基于路径的路由规则
----------------------

URI 路径经常被用来分发流量，例如，与带有host参数 `foo.com` 的请求，`/foo` 前缀应该路由到服务 `foo` ，而请求的路径是 `/bar` 应以 `ApisixRoute` 的方式路由到服务 `bar` ，配置如下：

```yaml
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: foor-bar-route
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

路径匹配又两种配置方式： `prefix` 和 `exact`， 默认为 `exact`，如果需要 `prefix`， 只需附加一个 `*`，
例如，`/id/*` 匹配前缀为 `/id/` 的所有路径

高级路由特性
-----------------------

基于路径的路由是最常见的，但如果还不够用，请尝试 `ApisixRoute` 中的其他路由功能，如 `methods`、`nginxVars`。

`methods` 根据 HTTP 方法分发流量， 以下配置路由请求使用 `GET` 方法创建 `foo` 服务（ Kubernetes Service ）。

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
      backend:
        serviceName: foo
        servicePort: 80
```

服务解析粒度
------------------------------

默认情况下，引用的服务都会被观察以便将最新端点列表更新到 Apache APISIX 。apisix-ingress-controller 提供了另一种机制，仅使用此服务的 `ClusterIP`，通过设置 `resolveGranularity` 就可以将流量解析到想要的 `service`（ 默认为 `endpoint` ）。

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
      backend:
        serviceName: foo
        servicePort: 80
        resolveGranularity: service
```

插件
-------

Apache APISIX 提供了40多个 [插件](https://github.com/apache/apisix/tree/master/docs/en/latest/plugins) ，可以被用在 `ApisixRoute` 中。 所有配置项的名称都与 APISIX 中的相同。

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
      backend:
        serviceName: foo
        servicePort: 80
      plugins:
        - name: cors
          enable: true
```

上述配置是对 host 为 `local.httpbin.org` 启用 [cors](https://github.com/apache/api6/blob/master/docs/en/latest/plugins/cors.md) 插件。
