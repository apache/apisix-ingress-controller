---
title: ApisixUpstream
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

ApisixUpstream 是 Kubernetes Service 的装饰者。 它的设计与 Kubernetes Service 相同，通过添加负载平衡、运行状况检查、重试、超时参数等。

借助于 `ApisixUpstream` 和 Kubernetes Service，apisix ingress controller 将生成 APISIX Upstream(s)。

要了解更多信息，请查看 [Apache APISIX 架构设计文档](https://github.com/apache/apisix/blob/master/doc/architecture-design.md#upstream) 。

### 配置负载均衡

为了合理地分散 Kubernetes Service 的请求，需要一个合适的负载平衡算法。

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  loadbalancer:
    type: ewma
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  selector:
    app: httpbin
  ports:
  - name: http
    port: 80
    targetPort: 8080
```

上例展示了使用[ewma](https://linkerd.io/2016/03/16/beyond-round-robin-load-balancing-for-latency/) 作为`httpbin` Service的负载均衡器.

有时候需要session sticky, 可以使用 [Consistent Hashing](https://en.wikipedia.org/wiki/Consistent_hashing) 负载均衡机制.

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  loadbalancer:
    type: chash
    hashOn: header
    key: "user-agent"
```

通过上面的设置，Apache APISIX 将跟跟 User-Agent header头来进行负载均衡.

### 配置健康检查

尽管 Kubelet 已经提供了 [probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#:~:text=The%20kubelet%20uses%20readiness%20probes,removed%20from%20Service%20load%20balancers.) 来检测 pod 是否正常， 你可能还需要更强大的健康检查机制， 比如被动反馈功能。

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  healthCheck:
    passive:
      unhealthy:
        httpCodes:
          - 500
          - 502
          - 503
          - 504
        httpFailures: 3
        timeout: 5s
    active:
      type: http
      httpPath: /healthz
      timeout: 5s
      host: www.foo.com
      healthy:
        successes: 3
        interval: 2s
        httpCodes:
          - 200
          - 206
```

上面的YAML代码段定义了一个被动健康检查器来检测的不健康状态的端点，一旦有三个连续的状态码（`500`, `502`, `503`, `504`）错误的请求，端点将被设置为不正常，并且在恢复正常之前，不能将任何请求路由到那里。

这就是为什么主动健康检查器会进来， 端点可能会关闭一段时间， 然后再次准备就绪， 主动健康检查器会连续检测这些不健康的端点，并将其拉入一旦满足正常条件（连续三个请求获得良好的状态代码，例如`200` and `206`）。

注意：主动健康检查器与liveness/readiness probes 有些重复，但如果使用被动反馈机制，则需要它。所以一旦你使用了 ApisixUpstream 的健康检查功能，活动运行状况检查器是必需的。

### 配置重试与超时

当请求发生故障（如暂时性网络错误或服务不可用）时，您可能希望代理重试，默认情况下重试计数为 `1` 。您可以通过指定 `retries` 字段来更改它。

以下配置将 `retries` 配置为 `3` ，这表示最多有 `3` 个请求发送到 Kubernetes 服务 `httpbin` 的端点。

应该记住的是将请求得不到任何答复就会不断的向下一个端点传递。也就是说，如果在响应的转移中发生错误或超时，修复它是不可能的。

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  retries: 3
```

默认的连接、读取和发送超时为 `60s`, 这可能不适合某些应用程序，只需在 `timeout` 字段中更改它们即可。

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  timeout:
    connect: 5s
    read: 10s
    send: 10s
```

上例中分别设置 connect, read 和 timeout 为 `5s`, `10s`, `10s`。

### 端口层级的设置

有时候，单个 Kubernetes 服务可能会公开多个端口，这些端口提供不同的功能，并且需要不同的上游配置。

在这种情况下，您可以为单个端口创建配置。

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: foo
spec:
  loadbalancer:
    type: roundrobin
  portLevelSettings:
  - port: 7000
    scheme: http
  - port: 7001
    scheme: grpc
---
apiVersion: v1
kind: Service
metadata:
  name: foo
spec:
  selector:
    app: foo
  portLevelSettings:
  - name: http
    port: 7000
    targetPort: 7000
  - name: grpc
    port: 7001
    targetPort: 7001
```

`foo`服务公开两个端口，其中一个使用 HTTP 协议，另一个使用grpc 协议。 与此同时，ApisixUpstream `foo` 设置了 `http` 协议端口为`7000` 和 `grpc`协议端口为`7001`（所有端口都是服务端口）。但是两个端口共享负载均衡配置。

如果服务定义多个端口但是只公开一个端口，`PortLevelSettings` 可以不用设置。
