---
title: AEP-0001 Gateway API support
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

## Summary

[Gateway API](https://github.com/kubernetes-sigs/gateway-api) is dedicated to achieving expressive and scalable Kubernetes service networking through many custom resources.

Apache APISIX Ingress controller can realize richer functions by adding support for Gateway API, including Gateway management, multi-cluster support and other features.

## Motivation

* Improve ease of use
* Support lifecycle management of Apache APISIX Gateway

### Goals

* Can bind the Apache APISIX Ingress controller with Gateway resources.
* The traffic rules defined by the Gateway API are processed by the Apache APISIX Gateway

### Non-Goals

* Supports all Gateway API versions and capabilities.

## Proposal

Add support from the definition of HTTP routing. Mainly cover the following resources:

* GatewayClass
* Gateway
* HTTPRoute
* TLSRoute
* ...

## Design Details

We need to add a separate switch for the Gateway API to control whether to enable this feature, and add corresponding controllers for various resources.

* `pkg/ingress/gateway_class.go`
* `pkg/ingress/gateway.go`
* `pkg/ingress/http_route.go`

These controllers can handle `gateway.networking.k8s.io/v1alpha2` version of `GatewayClass`, `Gateway` and `HTTPRoute` resources.

For real traffic definition rules, it needs to be translated into rules in Apache APISIX.

### GatewayClass controller

For `GatewayClass` resources, we need to have a unique identifier. We can define `controllerName = apisix.apache.org/gateway-controller` in the code.

```yaml
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: GatewayClass
metadata:
  name: apisix-lb
spec:
  controllerName: apisix.apache.org/gateway-controller
```

After the creation is successful, you will see the following results.

```bash
➜  ~ kubectl get gatewayclass
NAME        CONTROLLER                             AGE
apisix-lb   apisix.apache.org/gateway-controller   7m
```

We need to update its Status.

### Gateway controller

For the `Gateway` resource, we have two stages:

* Binding the existing Apache APISIX data plane;
* Create a self-managed Apache APISIX Gateway;

```yaml
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: Gateway
metadata:
  name: my-gateway
spec:
  gatewayClassName: apisix-lb
  listeners:
  - name: http
    protocol: HTTP
    port: 80
```

After correct processing, you will get the following results:

```bash
➜  ~ kubectl get gateway                
NAME         CLASS       ADDRESS   READY   AGE
my-gateway   apisix-lb   6.6.6.6   True    12m
```

### HTTPRoute controller

For the `HTTPRoute` resource, we need to complete its translation to APISIX.

```yaml
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: HTTPRoute
metadata:
  name: http-app-1
spec:
  parentRefs:
  - name: my-gateway
  hostnames:
  - "foo.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /bar
    backendRefs:
    - name: my-service1
      port: 8080
  - matches:
    - headers:
      - type: Exact
        name: magic
        value: foo
      queryParams:
      - type: Exact
        name: great
        value: example
      path:
        type: PathPrefix
        value: /some/thing
      method: GET
    backendRefs:
    - name: my-service2
      port: 8080
```

Need to create the corresponding route on Apache APISIX.

### TLSRoute Controller

TBD

### TCPRoute Controller

TBD

### UDPRoute Controller

TBD

### Test Plan

* Use the e2e test case cover examples in the above document.

### Graduation Criteria

TBD

## Production Readiness

* We already have perfect e2e coverage
* Gateway API reaches GA
