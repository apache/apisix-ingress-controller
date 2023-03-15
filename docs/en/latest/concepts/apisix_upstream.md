---
title: ApisixUpstream
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixUpstream
description: Guide to using ApisixUpstream custom Kubernetes resource.
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

`ApisixUpstream` is a Kubernetes CRD object that abstracts out a Kubernetes service and makes it richer by adding load balancing, health check, retry, and timeouts. It is designed to have the same name as the Kubernetes service.

See [reference](https://apisix.apache.org/docs/ingress-controller/references/apisix_upstream) for the full API documentation.

## Load balancing

The example below shows how you can configure load balacing in `ApisixUpstream` object using [ewma](https://linkerd.io/2016/03/16/beyond-round-robin-load-balancing-for-latency/):

```yaml
apiVersion: apisix.apache.org/v2
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

If you require sticky sessions, algorithms like [consistent hashing](https://en.wikipedia.org/wiki/Consistent_hashing) can be used for load balancing. The example below uses the `User-Agent` header for hashing:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  loadbalancer:
    type: chash
    hashOn: header
    key: "user-agent"
```

## Health check

kubelet provides [probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#:~:text=The%20kubelet%20uses%20readiness%20probes,removed%20from%20Service%20load%20balancers.) for health check. But if more features like passive feedback is required, a powerful health check mechanism is needed.

The example below shows how you can configure a passive health checker to detect unhealthy endpoints. Once there are three consecutive requests with the unhealthy status codes, the endpoint will be marked as unhealthy and requests will not be forwarded to it until it is healthy again.

The active health checker checks these unhealthy endpoints continuously for healthy status codes. Requests are forwarded to these endpoints again after they are healthy.

```yaml
apiVersion: apisix.apache.org/v2
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

:::note

The active health check configuration is mandatory if using the `healthCheck` feature in `ApisixUpstream`.

:::

## Retries and timeouts

You can configure APISIX to retry requests to tolerate network errors. By default, `retries` is `1`.

The example below configures `3` retries.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  retries: 3
```

:::note

If an error or timeout occurs while transferring a response to a client, it would not retry.

:::

You can also change the timeouts to fit your applications. The default connect, read, and send timeout is `60s`.

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  timeout:
    connect: 5s
    read: 10s
    send: 10s
```

## Port-level settings

A Kubernetes service can expose multiple ports to provide distinct functions (like different protocols). For each of the ports, a different Upstream configuration might be required.

In the example below, the `foo` service exposes two ports, one using HTTP and the other gRPC. The Upstream service is configured to use the correct scheme for the respective ports:

```yaml
apiVersion: apisix.apache.org/v2
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
  ports:
  - name: http
    port: 7000
    targetPort: 7000
  - name: grpc
    port: 7001
    targetPort: 7001
```
