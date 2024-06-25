---
title: ApisixRoute/v2beta3
keywords:
  - APISIX ingress
  - Apache APISIX
  - ApisixRoute
description: Reference for ApisixRoute/v2beta3 custom Kubernetes resource.
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

See [concepts](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_route) to learn more about how to use the ApisixRoute resource.

## Spec

See the [definition](https://github.com/apache/apisix-ingress-controller/blob/master/samples/deploy/crd/v1/ApisixRoute.yaml) on GitHub.

The table below describes each of the attributes in the spec. The fields `apiVersion`, `kind`, and `metadata` are similar to other Kubernetes resources and are excluded below.

| Attribute                            | Type               | Description                                                                                                                                                                                 |
|--------------------------------------|--------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| http                                 | array              | HTTP Route rules.                                                                                                                                                                           |
| http[].name                          | string (required)  | Route rule name.                                                                                                                                                                            |
| http[].priority                      | integer            | Route priority. Used to determined which Route to use when multiple routes contain the same URI. Large number means higher priority.                                                        |
| http[].match                         | object             | Conditions to match a request with the Route.                                                                                                                                               |
| http[].match.paths                   | array              | List of URIs to match the Route with. The Route will be used if any one of the URIs is matched.                                                                                             |
| http[].match.hosts                   | array              | List of hosts to match the Route with. The Route will be used if any one of the hosts is matched.                                                                                           |
| http[].match.methods                 | array              | List of HTTP methods (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, `CONNECT`, `TRACE`) to match the Route with. The Route will be used if any one of the methods is matched. |
| http[].match.remoteAddrs             | array              | List of IP addresses (CIDR format) to match the Route with. The Route will be used if any one of the IP address is matched.                                                                 |
| http[].match.exprs                   | array              | List of expressions to match the Route with. The Route will be used if any one of the expression is matched.                                                                                |
| http[].match.exprs[].subject         | object             | Subject for the expression.                                                                                                                                                                 |
| http[].match.exprs[].subject.scope   | string             | Scope of the subject. Can be one of `Header`, `Query`, `Cookie`, or `Path`.                                                                                                                 |
| http[].match.exprs[].subject.name    | string             | Subject name. Can be empty when the scope is `Path`.                                                                                                                                        |
| http[].match.exprs[].op              | string             | Operator for the expression. See [Expression operators](#expression-operators) for more details.                                                                                            |
| http[].match.exprs[].value           | string             | Value to compare the subject with. Can use either this or `http[].match.exprs[].set`.                                                                                                       |
| http[].match.exprs[].set             | array              | Set to compare the subject with. Only used when the operator is `In` or `NotIn`. Can use either this or `http[].match.exprs[].value`.                                                       |
| http[].backends                      | object             | List of backend services. If there are more than one, a weight based traffic split policy would be applied.                                                                                 |
| http[].backends[].serviceName        | string             | Name of the backend service. The service and the `ApisixRoute` resource should be created in the same namespace.                                                                            |
| http[].backends[].servicePort        | integer or string  | Port number or the name defined in the service object of the backend.                                                                                                                       |
| http[].backends[].resolveGranularity | string             | See [Service resolution granularity](#service-resolution-granularity) for details.                                                                                                          |
| http[].backends[].weight             | int                | Weight with which to split traffic to the backend. Defaults to `100` and is ignored when there is only one backend.                                                                         |
| http[].backends[].subset             | string             | Subset for the target service. Should be pre-defined in the `ApisixUpstream` resource.                                                                                                      |
| http[].plugins                       | array              | [APISIX Plugins](https://apisix.apache.org/docs/apisix/plugins/batch-requests/) to be executed if the Route is matched.                                                                     |
| http[].plugins[].name                | string             | Name of the Plugin. See [Plugin hub](https://apisix.apache.org/plugins/) for a list of available Plugins.                                                                                   |
| http[].plugins[].enable              | boolean            | When set to `true`, the Plugin is enabled on the Route.                                                                                                                                     |
| http[].plugins[].config              | object             | Configuration of the Plugin. Should have the same fields as in [APISIX docs](https://apisix.apache.org/docs/apisix/plugins/batch-requests/).                                                |
| http[].websocket                     | boolean            | When set to `true` enables websocket proxy.                                                                                                                                                 |
| stream                               | array              | Stream route rules. Contains TCP or UDP rules.                                                                                                                                              |
| stream[].protocol                    | string (required)  | The protocol of rule. Support `TCP` or `UDP`                                                                                                                                                |
| stream[].name                        | string (required)  | Name of the rule.                                                                                                                                                                           |
| stream[].match                       | object (required)  | Conditions to match the request with the Route.                                                                                                                                             |
| stream[].match.ingressPort           | integer (required) | Listening port in the Ingress proxy server. This port should be defined in the [APISIX configuration](https://github.com/apache/apisix/blob/master/conf/config-default.yaml#L101).          |
| stream[].backend                     | object             | Backend service (deprecated). Use `http[].backends` instead.                                                                                                                                |
| stream[].backend.serviceName         | string             | Name of the backend service (depricated). The service and the `ApisixRoute` resource should be created in the same namespace.                                                               |
| stream[].backend.servicePort         | integer or string  | Port number or the name defined in the service object of the backend (deprecated).                                                                                                          |
| stream[].backend.resolveGranularity  | string             | See [Service resolution granularity](#service-resolution-granularity) for details (depricated).                                                                                             |
| stream[].backend.subset              | string             | Subset for the target service (depricated). Should be pre-defined in the `ApisixUpstream` resource.                                                                                         |

## Expression operators

The following operators can be used in match expressions:

| Operator                     | Description                                                                     |
| ---------------------------- | ------------------------------------------------------------------------------- |
| Equal                        | Result of the `subject` should be equal to the `value`.                         |
| NotEqual                     | Result of the `subject` should not be equal to the `value`.                     |
| GreaterThan                  | Result of the `subject` should be a number and must be larger than the `value`. |
| LessThan                     | Result of the `subject` should be a number and must be less than the `value`.   |
| In                           | Result of the `subject` should be a part of the `set`.                          |
| NotIn                        | Result of the `subject` should be a part of the `set`.                          |
| RegexMatch                   | Result of the `subject` should match the PCRE regex pattern of the `value`.     |
| RegexNotMatch                | Result of the `subject` should not match the PCRE regex pattern of the `value`. |
| RegexMatchCaseInsensitive    | Similar to `RegexMatch` but case insensitive.                                   |
| RegexNotMatchCaseInsensitive | Similar to `RegexNotMatch` but case insensitive.                                |

## Service resolution granularity

By default, the service referenced will be watched to update its endpoint list in APISIX. To just use the `ClusterIP` of the service, you can set the `resolveGranularity` attribute to `service` (defaults to `endpoint`):

| Granularity | Description                                                                                                                                                |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| endpoint    | Upstream nodes are pods' IP adresses.                                                                                                                      |
| service     | Upstream nodes are service cluster IP. Load balancing is implemented by [kube-proxy](https://kubernetes.io/docs/concepts/overview/components/#kube-proxy). |
