---
title: Gateway API
keywords:
  - APISIX Ingress
  - Apache APISIX
  - Kubernetes Ingress
  - Gateway API
---
<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License. You may obtain a copy of the License at
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

Gateway API is dedicated to achieving expressive and scalable Kubernetes service networking through various custom resources.

By supporting Gateway API, the APISIX Ingress controller can realize richer functions, including Gateway management, multi-cluster support, and other features. It is also possible to manage running instances of the APISIX gateway through Gateway API resource management.

## Concepts

- **GatewayClass**: Defines a class of Gateways with a shared configuration and behavior. Each GatewayClass is managed by a single controller, although a controller may support multiple GatewayClasses.
- **Gateway**: Represents a request for network traffic handling within the cluster. A Gateway specifies how traffic enters the cluster and is directed to backend Services, typically by binding to one or more listeners.
- **HTTPRoute**: Configures routing for HTTP traffic.
- **GRPCRoute**: Configures routing for gRPC traffic.
- **ReferenceGrant**: Grants permission to reference resources across namespaces.
- **TLSRoute**: Defines routing rules for terminating or passing through TLS traffic.
- **TCPRoute**: Configures routing for TCP traffic.
- **UDPRoute**: Configures routing for UDP traffic.
- **BackendTLSPolicy**: Specifies how a Gateway should validate TLS connections to its backends, including trusted certificate authorities and verification modes.

## Gateway API Support Level

| Resource         | Core Support Level  | Extended Support Level | Implementation-Specific Support Level | API Version |
| ---------------- | ------------------- | ---------------------- | ------------------------------------- | ----------- |
| GatewayClass     | Supported           | N/A                    | Not supported                         | v1          |
| Gateway          | Partially supported | Partially supported    | Not supported                         | v1          |
| HTTPRoute        | Supported           | Partially supported    | Not supported                         | v1          |
| GRPCRoute        | Supported           | Supported              | Not supported                         | v1          |
| ReferenceGrant   | Supported           | Not supported          | Not supported                         | v1beta1     |
| TLSRoute         | Not supported       | Not supported          | Not supported                         | v1alpha2    |
| TCPRoute         | Supported           | Supported              | Not supported                         | v1alpha2    |
| UDPRoute         | Supported           | Supported              | Not supported                         | v1alpha2    |
| BackendTLSPolicy | Not supported       | Not supported          | Not supported                         | v1alpha3    |

## Examples

For configuration examples, see the Gateway API tabs in [Configuration Examples](../reference/example.md).

For a complete list of configuration options, refer to the [Gateway API Reference](https://gateway-api.sigs.k8s.io/reference/spec/). Be aware that some fields are not supported, or partially supported.

## Unsupported / Partially Supported Fields

The fields below are specified in the Gateway API specification but are either partially implemented or not yet supported in the APISIX Ingress Controller.

### HTTPRoute

| Fields                         | Status                 | Notes                                                                                   |
|--------------------------------|------------------------|-----------------------------------------------------------------------------------------|
| `spec.timeouts`                | Not supported          | The field is unsupported because ADC provides finer-grained timeout configuration (connect, read, write), whereas `spec.timeouts` only allows a general total timeout and upstream timeout, so it cannot be directly mapped. To configure route timeouts, you can use [BackendTrafficPolicy](../reference/api-reference.md#backendtrafficpolicyspec).  |
| `spec.retries`                 | Not supported          | The field is unsupported because APISIX does not support the features in retries. To configure route retries, you can use [BackendTrafficPolicy](../reference/api-reference.md#backendtrafficpolicyspec).  |
| `spec.sessionPersistence`      | Not supported          | APISIX does not support the configuration of cookie lifetimes. As an alternative, you can use [`chash` load balancer](../reference/api-reference.md#loadbalancer). |
| `spec.rules[].backendRefs[].filters[]` | Not supported | BackendRef-level filters are not implemented as data plane does not support filtering at this level; only rule-level filters (`spec.rules[].filters[]`) are supported. |

### Gateway

| Fields                                               | Status               | Notes                                                                                          |
|------------------------------------------------------|----------------------|------------------------------------------------------------------------------------------------|
| `spec.listeners[].port`               | Not supported*  | The configuration is required but ignored. This is due to limitations in the data plane: it cannot dynamically open new ports. Since the Ingress Controller does not manage the data plane deployment, it cannot automatically update the configuration or restart the data plane to apply port changes.    |
| `spec.listeners[].allowedRoutes.kinds`               | Partially supported  | Only `HTTPRoute` (group `gateway.networking.k8s.io`) is accepted; other kinds are rejected.    |
| `spec.listeners[].tls.certificateRefs[].group` | Partially supported | Only `""` is supported; other group values cause validation failure. |
| `spec.listeners[].tls.certificateRefs[].kind`        | Partially supported  | Only `Secret` is supported.                                                                    |
| `spec.listeners[].tls.mode`                          | Partially supported  | `Terminate` is implemented; `Passthrough` is effectively unsupported for Gateway listeners.    |
| `spec.addresses`                                     | Not supported        | Controller does not read or act on `spec.addresses`.                                           |
