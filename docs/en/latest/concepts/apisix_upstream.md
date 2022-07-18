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

ApisixUpstream is the decorator of Kubernetes Service. It's designed to have same name with its target Kubernetes Service, it makes the Kubernetes Service richer by adding
load balancing, health check, retry, timeout parameters and etc.

Resort to `ApisixUpstream` and the Kubernetes Service, apisix ingress controller will generates the APISIX Upstream(s).
To learn more, please check the [Apache APISIX architecture-design docs](https://github.com/apache/apisix/blob/master/docs/en/latest/terminology/upstream.md).

### Configuring Load Balancer

A proper load balancing algorithm is required to scatter requests reasonably for a Kubernetes Service.

```yaml
apiVersion: apisix.apache.org/v2beta3
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

The above example shows that [ewma](https://linkerd.io/2016/03/16/beyond-round-robin-load-balancing-for-latency/) is used as the load balancer for Service `httpbin`.

Sometimes the session sticky is desired, and you can use the [Consistent Hashing](https://en.wikipedia.org/wiki/Consistent_hashing) load balancing algorithm.

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  loadbalancer:
    type: chash
    hashOn: header
    key: "user-agent"
```

With the above settings, Apache APISIX will distributes requests according to the User-Agent header.

### Configuring Health Check

Although Kubelet already provides [probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#:~:text=The%20kubelet%20uses%20readiness%20probes,removed%20from%20Service%20load%20balancers.) to detect whether pods are healthy, you may still need more powerful health check mechanism,
like the passive feedback capability.

```yaml
apiVersion: apisix.apache.org/v2beta3
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

The above YAML snippet defines a passive health checker to detect the unhealthy state for
endpoints, once there are three consecutive requests with bad status code (one of `500`, `502`, `503`, `504`), the endpoint
will be set to unhealthy and no requests can be routed there until it's healthy again.

That's why the active health checker comes in, endpoints might be down for a short while and ready again, the active health checker detects these unhealthy endpoints continuously, and pull them
up once the healthy conditions are met (three consecutive requests got good status codes, e.g. `200` and `206`).

Note the active health checker is somewhat duplicated with the liveness/readiness probes but it's required if the passive feedback mechanism is in use. So once you use the health check feature in ApisixUpstream,
the active health checker is mandatory.

### Configuring Retry and Timeout

You may want the proxy to retry when requests occur faults like transient network errors
or service unavailable, by default the retry count is `1`. You can change it by specifying the `retries` field.

The following configuration configures the `retries` to `3`, which indicates there'll be at most `3` requests sent to
Kubernetes service `httpbin`'s endpoints.

One should bear in mind that passing a request to the next endpoint is only possible
if nothing has been sent to a client yet. That is, if an error or timeout occurs in the middle
of the transferring of a response, fixing this is impossible.

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  retries: 3
```

The default connect, read and send timeout are `60s`, which might not proper for some applications,
just change them in the `timeout` field.

```yaml
apiVersion: apisix.apache.org/v2beta3
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  timeout:
    connect: 5s
    read: 10s
    send: 10s
```

The above examples sets the connect, read and timeout to `5s`, `10s`, `10s` respectively.

### Port Level Settings

Once in a while a single Kubernetes Service might expose multiple ports which provides distinct functions and different Upstream configurations are required.
In that case, you can create configurations for individual port.

```yaml
apiVersion: apisix.apache.org/v2beta3
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

The `foo` service exposes two ports, one of them use HTTP protocol and the other uses grpc protocol.
In the meanwhile, the ApisixUpstream `foo` sets `http` scheme for port `7000` and `grpc` scheme for `7001`
(all ports are the service port). But both ports shares the load balancer configuration.

`PortLevelSettings` is not mandatory if the service only exposes one port but is useful when multiple ports are defined.
