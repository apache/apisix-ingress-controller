---
title: ApisixUpstream
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixUpstream
description: Reference for ApisixUpstream custom Kubernetes resource.
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

See [concepts](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_upstream) to learn more about how to use the ApisixUpstream resource.

## Spec

See the [definition](https://github.com/apache/apisix-ingress-controller/blob/master/samples/deploy/crd/v1/ApisixUpstream.yaml) on GitHub.

| Attribute                                  | Type              | Description                                                                                                                                                                                                                      |
|--------------------------------------------|-------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| scheme                                     | string            | Scheme/Protocol used to talk to the Upstream service. Can be one of `http`, `https`, `grpc`, `grpcs`. Defaults to `http`.                                                                                                        |
| loadbalancer                               | object            | Load balancer configuration.                                                                                                                                                                                                     |
| loadbalancer.type                          | string            | Load balancing algorithm to use. Can be one of `roundrobin`, `ewma`, `least_conn`, or `chash`. Defaults to `roundrobin`.                                                                                                         |
| loadbalancer.hashOn                        | string            | Value to hash on. Can only be used if the `type` is `chash`. Can be one of `vars`, `header`, `vars_combinations`, `cookie`, and `consumers`. Defaults to `vars`.                                                                 |
| loadbalancer.key                           | string            | Hash key. Can only be used if the `type` is `chash`.                                                                                                                                                                             |
| retries                                    | int               | Number of retries while passing the request to the Upstream.                                                                                                                                                                     |
| timeout                                    | object            | Timeouts for connecting, sending, and receiving messages between Ingress and the service.                                                                                                                                        |
| timeout.connect                            | string            | Connect timeout in the form "72h3m0.5s".                                                                                                                                                                                         |
| timeout.read                               | string            | Read timeout in the form "72h3m0.5s".                                                                                                                                                                                            |
| timeout.send                               | string            | Send timeout in the form "72h3m0.5s".                                                                                                                                                                                            |
| healthCheck                                | object            | Configures the parameters of the [health check](https://apisix.apache.org/docs/apisix/tutorials/health-check/).                                                                                                                  |
| healthCheck.active                         | object            | Active health check configuration. Required if configuring health check.                                                                                                                                                         |
| healthCheck.active.type                    | string            | Health check type. Can be one of `http`, `https`, or `tcp`. Defaults to `http`.                                                                                                                                                  |
| healthCheck.active.timeout                 | string            | Timeout in the form "72h3m0.5s". Defaults to `1s`.                                                                                                                                                                               |
| healthCheck.active.concurrency             | int               | Number of probes that can be sent simultaneously. Defaults to `10`.                                                                                                                                                              |
| healthCheck.active.host                    | string            | Host header in the HTTP probe request. Valid only if the health check type is `http` or `https`.                                                                                                                                 |
| healthCheck.active.port                    | int               | Target port to receive probes. It is required to specify this attribute if the health check service exposes different ports. Note that the port is the container port and not the service port.                                  |
| healthCheck.active.httpPath                | string            | URI in the HTTP probe request. Valid only if the health check type is `http` or `https`.                                                                                                                                         |
| healthCheck.active.strictTLS               | boolean           | When set to `true` enables the strict TLS mode. Valid only if the health check type is `https`. Defaults to `true`.                                                                                                              |
| healthCheck.active.requestHeaders          | array of strings  | Additional HTTP request headers carried in the HTTP probe. Valid only if the health check type is `http` or `https`.                                                                                                             |
| healthCheck.active.healthy                 | object            | Conditions to check to see if an endpoint is healthy.                                                                                                                                                                            |
| healthCheck.active.healthy.successes       | int               | Number of consecutive successful requests before an endpoint is set as healthy. By default set to `2`.                                                                                                                           |
| healthCheck.active.healthy.httpCodes       | array of integers | Status codes that will indicate an endpoint is healthy. Valid only if the health check type is `http` or `https`. Defaults to `[200, 302]`.                                                                                      |
| healthCheck.active.healthy.interval        | string            | Send interval for the probes in the form "72h3m0.5s".                                                                                                                                                                            |
| healthCheck.active.unhealthy               | object            | Conditions to check to see if an endpoint is unhealthy.                                                                                                                                                                          |
| healthCheck.active.unhealthy.httpFailures  | int               | Number of consecutive unsuccessful HTTP requests before an endpoint is set as unhealthy. Valid only if the health check type is `http` or `https`. By default set to `5`.                                                        |
| healthCheck.active.unhealthy.tcpFailures   | int               | Number of consecutive unsuccessful TCP requests before an endpoint is set as unhealthy. Valid only if the health check type is `tcp`. By default set to `2`.                                                                     |
| healthCheck.active.unhealthy.httpCodes     | array of integers | Status codes that will indicate an endpoint is unhealthy. Valid only if the health check type is `http` or `https`. Defaults to `[429, 404, 500, 501, 502, 503, 504, 505]`.                                                      |
| healthCheck.active.unhealthy.interval      | string            | Send interval for the probes in the form "72h3m0.5s".                                                                                                                                                                            |
| healthCheck.passive                        | object            | Passive health check configuration.                                                                                                                                                                                              |
| healthCheck.passive.type                   | string            | Health check type. Can be one of `http`, `https`, or `tcp`. Defaults to `http`.                                                                                                                                                  |
| healthCheck.passive.healthy                | object            | Conditions to check to see if an endpoint is healthy.                                                                                                                                                                            |
| healthCheck.passive.healthy.successes      | int               | Number of consecutive successful requests before an endpoint is set as healthy. By default set to `5`.                                                                                                                           |
| healthCheck.passive.healthy.httpCodes      | array of integers | Status codes that will indicate an endpoint is healthy. Valid only if the health check type is `http` or `https`. Defaults to `[200, 201, 202, 203, 204, 205, 206, 207, 208, 226, 300, 301, 302, 303, 304, 305, 306, 307, 308]`. |
| healthCheck.passive.unhealthy              | object            | Conditions to check to see if an endpoint is unhealthy.                                                                                                                                                                          |
| healthCheck.passive.unhealthy.httpFailures | int               | Number of consecutive unsuccessful HTTP requests before an endpoint is set as unhealthy. Valid only if the health check type is `http` or `https`. By default set to `5`.                                                        |
| healthCheck.passive.unhealthy.tcpFailures  | int               | Number of consecutive unsuccessful TCP requests before an endpoint is set as unhealthy. Valid only if the health check type is `tcp`. By default set to `2`.                                                                     |
| healthCheck.passive.unhealthy.httpCodes    | array of integers | Status codes that will indicate an endpoint is unhealthy. Valid only if the health check type is `http` or `https`. Defaults to `[429, 404, 500, 501, 502, 503, 504, 505]`.                                                      |
| portLevelSettings                          | array             | Settings for individual ports.                                                                                                                                                                                                   |
| portLevelSettings.port                     | int               | Valid port number defined in the Kubernetes service.                                                                                                                                                                             |
| portLevelSettings.scheme                   | string            | Scheme to use on the specific port. Will override the global `scheme` attribute.                                                                                                                                                 |
| portLevelSettings.loadbalancer             | object            | Load balancer to use on the specific port. Will override the global `loadbalancer` attribute.                                                                                                                                    |
| portLevelSettings.healthCheck              | object            | Health check configuration on the specific port. Will override the global `healthCheck` attribute.                                                                                                                               |
| subsets                                    | array             | List of service subsets. Use pod labels to organize service endpoints to different groups.                                                                                                                                       |
| subsets[].name                             | string            | Name of the subset.                                                                                                                                                                                                              |
| subsets[].labels                           | object            | Label map of the subset.                                                                                                                                                                                                         |
| discovery                                  | object            | Discovery is used to configure Service Discovery for upstream.                                                                                                                                                                   |
| discovery.serviceName                      | string            | Name of the upstream service.                                                                                                                                                                                                    |
| discovery.type                             | string            | Types of Service Discovery, which indicates what registry in APISIX the discovery uses. Should match the entry in APISIX's config. Can refer to the [doc](https://apisix.apache.org/docs/apisix/discovery/)                                                                           |
| discovery.args                             | object            | Args map for discovery-spcefic parameters. Also can refer to the [doc](https://apisix.apache.org/docs/apisix/discovery/)                                                                                                         |
| passHost                                   | string            | Configures the host when the request is forwarded to the upstream. Can be one of pass, node or rewrite. Defaults to pass if not specified: pass - transparently passes the client's host to the Upstream, node - uses the host configured in the node of the Upstream, rewrite - uses the value configured in upstreamHost.
| upstreamHost                               | string            | Specifies the host of the Upstream request. This is only valid if the passHost is set to rewrite.
