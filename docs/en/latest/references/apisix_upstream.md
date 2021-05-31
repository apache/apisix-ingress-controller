---
title: ApisixUpstream Reference
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
| healthCheck | object | The health check parameters, see [Health Check](https://github.com/apache/apisix/blob/master/docs/en/latest/health-check.md) for more details. |
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
| subsets | array | service subset list, use pod labels to organize service endpoints to different groups. |
| subsets[].name | string | the subset name. |
| subsets[].labels | object | the subset label map. |
