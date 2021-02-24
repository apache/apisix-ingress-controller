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

# CRD specification

In order to control the behavior of the proxy ([Apache APISIX](https://github.com/apache/apisix)), the following CRDs should be defined.

## CRD Types

- [ApisixRoute](#apisixroute)
- [ApisixUpstream](#apisixupstream)
  - [Configuring Load Balancer](#configuring-load-balancer)
  - [Configuring Health Check](#configuring-load-balancer)
  - [Configuring Retry and Timeout](#configuring-retry-and-timeout)
  - [Port Level Settings](#port-level-settings)
  - [Configuration References](#configuration-references)
- [ApisixTls](#apisixtls)

## ApisixRoute

`ApisixRoute` corresponds to the `Route` object in Apache APISIX. The `Route` matches the client's request by defining rules,
then loads and executes the corresponding plugin based on the matching result, and forwards the request to the specified Upstream.
To learn more, please check the [Apache APISIX architecture-design docs](https://github.com/apache/apisix/blob/master/doc/architecture-design.md#route).

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  name: httpserverRoute
  namespace: cloud
spec:
  rules:
  - host: test.apisix.apache.org
    http:
      paths:
      - backend:
          serviceName: httpserver
          servicePort: 8080
        path: /hello*
        plugins:
          - name: limit-count
            enable: true
            config:
              count: 2
              time_window: 60
              rejected_code: 503
              key: remote_addr
```

|     Field     |  Type    |                    Description                     |
|---------------|----------|----------------------------------------------------|
| rules         | array    | ApisixRoute's request matching rules.              |
| host          | string   | The requested host.                                |
| http          | object   | Route rules are applied to the scope of layer 7 traffic.     |
| paths         | array    | Path-based `route` rule matching.                     |
| backend       | object   | Backend service information configuration.         |
| serviceName   | string   | The name of backend service. `namespace + serviceName + servicePort` form an unique identifier to match the back-end service.                      |
| servicePort   | int      | The port of backend service. `namespace + serviceName + servicePort` form an unique identifier to match the back-end service.                      |
| path          | string   | The URI matched by the route. Supports exact match and prefix match. Example，exact match: `/hello`, prefix match: `/hello*`.                     |
| plugins       | array    | Custom plugin collection (Plugins defined in the `route` level). For more plugin information, please refer to the [Apache APISIX plugin docs](https://github.com/apache/apisix/tree/master/doc/plugins).    |
| name          | string   | The name of the plugin. For more information about the example plugin, please check the [limit-count docs](https://github.com/apache/apisix/blob/master/doc/plugins/limit-count.md#Attributes).             |
| enable        | boolean  | Whether to enable the plugin, `true`: means enable, `false`: means disable.      |
| config        | object   | Configuration of plugin information. Note: The check of configuration schema is missing now, so please be careful when editing.    |

**Support partial `annotation`**

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  annotations:
    k8s.apisix.apache.org/ingress.class: apisix_group
    k8s.apisix.apache.org/ssl-redirect: 'false'
    k8s.apisix.apache.org/whitelist-source-range:
      - 1.2.3.4/16
      - 4.3.2.1/8
  name: httpserverRoute
  namespace: cloud
spec:
```

|         Field                                  |    Type    |                       Description                                  |
|------------------------------------------------|------------|--------------------------------------------------------------------|
| `k8s.apisix.apache.org/ssl-redirect`           | boolean    | Whether to force http redirect to https. `ture`: means to force conversion to https, `false`: means not to convert.   |
| `k8s.apisix.apache.org/ingress.class`          | string     | Grouping of ingress.                                               |
| `k8s.apisix.apache.org/whitelist-source-range` | array      | Whitelist of IPs allowed to be accessed.                           |

## ApisixUpstream

ApisixUpstream is the decorator of Kubernetes Service. It's designed to have same name with its target Kubernetes Service, it makes the Kubernetes Service richer by adding
load balancing, health check, retry, timeout parameters and etc.

Resort to `ApisixUpstream` and the Kubernetes Service, apisix ingress controller will generates the APISIX Upstream(s).
To learn more, please check the [Apache APISIX architecture-design docs](https://github.com/apache/apisix/blob/master/doc/architecture-design.md#upstream).

### Configuring Load Balancer

A proper load balancing algorithm is required to scatter requests reasonably for a Kubernetes Service.

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

The above example shows that [ewma](https://linkerd.io/2016/03/16/beyond-round-robin-load-balancing-for-latency/) is used as the load balancer for Service `httpbin`.

Sometimes the session sticky is desired, and you can use the [Consistent Hashing](https://en.wikipedia.org/wiki/Consistent_hashing) load balancing algorithm.

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

With the above settings, Apache APISIX will distributes requests according to the User-Agent header.

### Configuring Health Check

Although Kubelet already provides [probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#:~:text=The%20kubelet%20uses%20readiness%20probes,removed%20from%20Service%20load%20balancers.) to detect whether pods are healthy, you may still need more powerful health cheak mechanism,
like the passive feedback capability.

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

The above YAML snippet defines a passive health checker to detech the unhealthy state for
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
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: httpbin
spec:
  retries: 3
```

The default connect, read and send timeout are `60s`, which might not proper for some applicartions,
just change them in the `timeout` field.

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

The above examples sets the connect, read and timeout to `5s`, `10s`, `10s` respectively.

### Port Level Settings

Once in a while a single Kubernetes Service might expose multiple ports which provides distinct functions and different Upstream configurations are required.
In that case, you can create configurations for individual port.

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

The `foo` service exposes two ports, one of them use HTTP protocol and the other uses grpc protocol.
In the meanwhile, the ApisixUpstream `foo` sets `http` scheme for port `7000` and `grpc` scheme for `7001`
(all ports are the service port). But both ports shares the load balancer configuration.

`PortLevelSettings` is not mandatory if the service only exposes one port but is useful when multiple ports are defined.

### Configuration References

|     Field     |  Type    | Description    |
|---------------|----------|----------------|
| scheme        | string   | The protocol used to talk to the Service, can be `http`, `grpc`, default is `http`.   |
| loadbalancer  | object   | The load balancing algorithm of this upstream service |
| loadbalancer.type | string | The load balancing type, can be `roundrobin`, `ewma`, `least_conn`, `chash`, default is `roundrobin`. |
| loadbalancer.hashOn | string | The hash value source scope, only take effects if the `chash` algorithm is in use. Values can `vars`, `header`, `vars_combinations`, `cookie` and `consumers`, default is `vars`. |
| loadbalancer.key | string | The hash key, only in valid if the `chash` algorithm is used.
| retries | int | The retry count. |
| timeout | object | The timeout settings. |
| timeout.connect | time duration in the form "72h3m0.5s" | The connect timeout. |
| timeout.read | time duration in the form "72h3m0.5s" | The read timeout. |
| timeout.send | time duration in the form "72h3m0.5s" | The send timeout. |
| healthCheck | object | The health check parameters, see [Health Check](https://github.com/apache/apisix/blob/master/doc/health-check.md) for more details. |
| healthCheck.active | object | active health check configuration, which is a mandatory field. |
| healthCheck.active.type | string | health check type, can be `http`, `https` and `tcp`, default is `http`. |
| healthCheck.active.timeout | time duration in the form "72h3m0.5s" | the timeout settings for the probe, default is `1s`. |
| healthCheck.active.concurrency | int | how many probes can be sent simultaneously, default is `10`. |
| healthCheck.active.host | string | host header in http probe request, only in valid if the active health check type is `http` or `https`. |
| healthCheck.active.port | int | target port to receive probes, it's necessary to specify this field if the health check service exposes by different port, note the port value here is the container port, not the service port. |
| healthCheck.active.httpPath | string | the HTTP URI path in http probe, only in valid if the active health check type is `http` or `https`. |
| healthCheck.active.strictTLS | boolean | whether to use the strict mode when use TLS, only in valid if the active health check type is `https`, default is `true`. |
| healthCheck.active.requestHeaders | array of string | Extra HTTP requests carried in the http probe, only in valid if the active health check type is `http` or `https`. |
| healthCheck.active.healthy | object | The conditions to judge an endpoint is healthy. |
| healthCheck.active.healthy.successes | int | The number of consecutive requests needed to set an endpoint as healthy, default is `2`. |
| healthCheck.active.healthy.httpCodes | array of integer | Good status codes list to check whether a probe is successful, only in valid if the active health check type is `http` or `https`, default is `[200, 302]`. |
| healthCheck.active.healthy.interval | time duration in the form "72h3m0.5s" | The probes sent interval (for healthy endpoints). |
| healthCheck.active.unhealthy | object | The conditions to judge an endpoint is unhealthy. |
| healthCheck.active.unhealthy.httpFailures | int | The number of consecutive http requests needed to set an endpoint as unhealthy, only in valid if the active health check type is `http` or `https`, default is `5`. |
| healthCheck.active.unhealthy.tcpFailures | int | The number of consecutive tcp connections needed to set an endpoint as unhealthy, only in valid if the active health check type is `tcp`, default is `2`. |
| healthCheck.active.unhealthy.httpCodes | array of integer | Bad status codes list to check whether a probe is failed, only in valid if the active health check type is `http` or `https`, default is `[429, 404, 500, 501, 502, 503, 504, 505]`. |
| healthCheck.active.unhealthy.interval | time duration in the form "72h3m0.5s" | The probes sent interval (for unhealthy endpoints). |
| healthCheck.passive | object | passive health check configuration, which is an optional field. |
| healthCheck.passive.type | string | health check type, can be `http`, `https` and `tcp`, default is `http`. |
| healthCheck.passive.healthy | object | The conditions to judge an endpoint is healthy. |
| healthCheck.passive.healthy.successes | int | The number of consecutive requests needed to set an endpoint as healthy, default is `5`. |
| healthCheck.passive.healthy.httpCodes | array of integer | Good status codes list to check whether a probe is successful, only in valid if the active health check type is `http` or `https`, default is `[200, 201, 202, 203, 204, 205, 206, 207, 208, 226, 300, 301, 302, 303, 304, 305, 306, 307, 308]`. |
| healthCheck.passive.unhealthy | object | The conditions to judge an endpoint is unhealthy. |
| healthCheck.passive.unhealthy.httpFailures | int | The number of consecutive http requests needed to set an endpoint as unhealthy, only in valid if the active health check type is `http` or `https`, default is `5`. |
| healthCheck.passive.unhealthy.tcpFailures | int | The number of consecutive tcp connections needed to set an endpoint as unhealthy, only in valid if the active health check type is `tcp`, default is `2`. |
| healthCheck.passive.unhealthy.httpCodes | array of integer | Bad status codes list to check whether a probe is failed, only in valid if the active health check type is `http` or `https`, default is `[429, 404, 500, 501, 502, 503, 504, 505]`. |
| portLevelSettings | array | Settings for each individual port. |
| portLevelSettings.port | int | The port number defined in the Kubernetes Service, must be a valid port. |
| portLevelSettings.scheme | string | same as `scheme` but takes higher precedence. |
| portLevelSettings.loadbalancer | object | same as `loadbalancer` but takes higher precedence. |
| portLevelSettings.healthCheck | object | same as `healthCheck` but takes higher precedence. |

## ApisixTls

`ApisixTls` corresponds to the SSL load matching route in Apache APISIX.
To learn more, please check the [Apache APISIX architecture-design docs](https://github.com/apache/apisix/blob/master/doc/architecture-design.md#router).

Structure example:

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixSSL
metadata:
  name: duiopen
spec：
  hosts:
  - asr.duiopen.com
  - tts.duiopen.com
  secret:
    name: all.duiopen.com
    namespace: cloud
```

|     Field     |  Type    | Description                     |
|---------------|----------|---------------------------------|
| hosts         | array    | The domain list to identify which hosts (matched with SNI) can use the TLS certificate stored in the Secret.  |
| secret        | object   | The definition of the related Secret object with current ApisixTls object.                               |
| name          | string   | The name of secret, the secret contains key and cert for `TLS`.       |
| namespace     | string   | The namespace of secret , the secret contains key and cert for `TLS`.  |

[Back to top](#crd-types)
