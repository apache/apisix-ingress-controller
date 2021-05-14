---
title: ApisixRoute/v2alpha1 Reference
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

## Spec

Meaning of each field in the spec of ApisixRoute are followed, the top level fields (`apiVersion`, `kind` and `metadata`) are same as other Kubernetes Resources.

|     Field     |  Type    |                    Description                     |
|---------------|----------|----------------------------------------------------|
| http         | array    | ApisixRoute's HTTP route rules.              |
| http[].name          | string (required)  | The route rule name.                                |
| http[].priority          | integer   | The route priority, it's used to determine which route will be hitted when multile routes contains the same URI. Large number means higher priority.     |
| http[].match         | object    | Route match conditions.                     |
| http[].match.paths       | array   | A series of URI that should be matched (oneof) to use this route rule.         |
| http[].match.hosts   | array   | A series of hosts that should be matched (oneof) to use this route rule.
| http[].match.methods | array | A series of HTTP methods(`GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, `CONNECT`, `TRACE`) that should be matched (oneof) to use this route rule.
| http[].match.remoteAddrs   | array      | A series of IP address (CIDR format) that should be matched (oneof) to use this route rule.
| http[].match.exprs          | array   | A series expressions that the results should be matched (oneof) to use this route rule.
| http[].match.exprs[].subject       | object    | Expression subject.
| http[].match.exprs[].subject.scope       | string    | Specify where to find the subject, values can be `Header`, `Query`, `Cookie` and `Path`.
| http[].match.exprs[].subject.name       | string    | Specify subject name, when scope is `Path`, this field can be absent.
| http[].match.exprs[].op | string | Expression operator, see [Expression Operators](#expression-operators) for the detail of enumerations.
| http[].match.exprs[].value | string | Expected expression result, it's exclusive with `http[].match.exprs[].set`.
| http[].match.exprs[].set | array | Expected expression result set, only used when the operator is `In` or `NotIn`, it's exclusive with `http[].match.exprs[].value`.
| http[].backend | object | The backend service. Deprecated: use http[].backends instead.
| http[].backend.serviceName | string | The backend service name, note the service and ApisixRoute should be created in the same namespace. Cross namespace referencing is not allowed.
| http[].backend.servicePort | integer or string | The backend service port, can be the port number or the name defined in the service object.
| http[].backend.resolveGranularity | string | See [Service Resolve Granularity](#service-resolve-granularity) for the details.
| http[].backends | object | The backend services. When the number of backends more than one, weight based traffic split policy will be applied to shifting traffic between these backends.
| http[].backends[].serviceName | string | The backend service name, note the service and ApisixRoute should be created in the same namespace. Cross namespace referencing is not allowed.
| http[].backends[].servicePort | integer or string | The backend service port, can be the port number or the name defined in the service object.
| http[].backends[].resolveGranularity | string | See [Service Resolve Granularity](#service-resolve-granularity) for the details.
| http[].backends[].weight | int | The backend weight, which is critical when shifting traffic between multiple backends, default is `100`. Weight is ignored when there is only one backend.
| http[].plugins | array | A series of APISIX plugins that will be executed once this route rule is matched |
| http[].plugins[].name | string | The plugin name, see [docs](http://apisix.apache.org/docs/apisix/getting-started) for learning the available plugins.
| http[].plugins[].enable | boolean | Whether the plugin is in use |
| http[].plugins[].config | object | The plugin configuration, fields should be same as in APISIX. |
| http[].websocket | boolean | Whether enable websocket proxy. |
| tcp | array | ApisixRoutes' tcp route rules. |
| tcp[].name | string (required) | The Route rule name. |
| tcp[].match | object (required) | The Route match conditions. |
| tcp[].match.ingressPort | integer (required) | the Ingress proxy server listening port, note since APISIX doesn't support dynamic listening, this port should be defined in [apisix configuration](https://github.com/apache/apisix/blob/master/conf/config-default.yaml#L101).|
| tcp[].backend | object | The backend service. Deprecated: use http[].backends instead.
| tcp[].backend.serviceName | string | The backend service name, note the service and ApisixRoute should be created in the same namespace. Cross namespace referencing is not allowed.
| tcp[].backend.servicePort | integer or string | The backend service port, can be the port number or the name defined in the service object.
| tcp[].backend.resolveGranularity | string | See [Service Resolve Granularity](#service-resolve-granularity) for the details.

## Expression Operators

| Operator | Meaning |
|----------|---------|
| Equal| The result of `subject` should be equal to the `value` |
| NotEqual | The result of `subject` should not be equal to `value` |
| GreaterThan | The result of `subject` should be a number and it must larger then `value`. |
| LessThan | The result of `subject` should be a number and it must less than `value`. |
| In | The result of `subject` should be inside the `set`. |
| NotIn | The result of `subject` should not be inside the `set`. |
| RegexMatch | The result of `subject` should be matched by the `value` (a PCRE regex pattern). |
| RegexNotMatch | The result of `subject` should not be matched by the `value` (a PCRE regex pattern). |
| RegexMatchCaseInsensitive | Similar with `RegexMatch` but the match process is case insensitive |
| RegexNotMatchCaseInsensitive | Similar with `RegexNotMatchCaseInsensitive` but the match process is case insensitive. |

## Service Resolve Granularity

The service resolve granularity determines whether the [Serivce ClusterIP](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types) or its endpoints should be filled in the target upstream of APISIX.

| Granularity | Meaning |
| ----------- | ------- |
| endpoint | Filled upstream nodes by Pods' IP.
| service | Filled upstream nodes by Service ClusterIP, in such a case, loadbalacing are implemented by [kube-proxy](https://kubernetes.io/docs/concepts/overview/components/#kube-proxy).|
